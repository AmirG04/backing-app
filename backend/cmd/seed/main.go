// cmd/seed carga un archivo JSON de datos de prueba (usuarios, cuentas,
// transacciones) directamente a Postgres y TigerBeetle, para pruebas de
// volumen. Es un programa independiente del servidor API - no se ejecuta
// en produccion, solo se corre manualmente cuando se necesita poblar la
// base de datos.
//
// Soporta multiples cuentas por usuario (tabla accounts, 1 usuario -> N
// cuentas), igual que el resto del backend.
//
// Uso: ./seed /ruta/al/archivo.json
package main

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"

	tbtypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"

	"banking-app/backend/internal/config"
	"banking-app/backend/internal/db"
)

type seedUser struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	FullName  string `json:"full_name"`
	CreatedAt string `json:"created_at"`
}

type seedAccount struct {
	AccountNumber  string  `json:"account_number"`
	UserID         string  `json:"user_id"`
	InitialBalance float64 `json:"initial_balance"`
	Currency       string  `json:"currency"`
	AccountType    string  `json:"account_type"`
}

type seedTransaction struct {
	FromAccount string  `json:"from_account"`
	ToAccount   string  `json:"to_account"`
	Amount      float64 `json:"amount"`
}

type seedData struct {
	Users        []seedUser        `json:"users"`
	Accounts     []seedAccount     `json:"accounts"`
	Transactions []seedTransaction `json:"transactions"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("uso: seed <ruta-al-archivo-json>")
	}
	path := os.Args[1]

	raw, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("error leyendo archivo: %v", err)
	}
	var data seedData
	if err := json.Unmarshal(raw, &data); err != nil {
		log.Fatalf("error parseando json: %v", err)
	}
	log.Printf("cargados del archivo: %d usuarios, %d cuentas, %d transacciones",
		len(data.Users), len(data.Accounts), len(data.Transactions))

	cfg := config.Load()

	pgPool, err := db.NewPostgresPool(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("error conectando a postgres: %v", err)
	}
	defer pgPool.Close()
	if err := db.Migrate(pgPool); err != nil {
		log.Fatalf("error migrando: %v", err)
	}

	clusterID, err := strconv.ParseUint(cfg.TigerBeetleCluster, 10, 64)
	if err != nil {
		log.Fatalf("TIGERBEETLE_CLUSTER_ID invalido: %v", err)
	}
	tbClient, err := db.NewTigerBeetleClient(clusterID, []string{cfg.TigerBeetleAddress})
	if err != nil {
		log.Fatalf("error conectando a tigerbeetle: %v", err)
	}
	defer tbClient.Close()
	if err := tbClient.EnsureBankAccount(); err != nil {
		log.Fatalf("error inicializando cuenta del banco: %v", err)
	}

	ctx := context.Background()

	// 1. Crear usuarios (sin cuentas todavia - eso va en el paso 2).
	userExists := map[string]bool{}
	usersCreated, usersSkipped := 0, 0
	for i, u := range data.Users {
		hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("aviso: error hasheando password de %s: %v", u.Email, err)
			usersSkipped++
			continue
		}

		createdAt, err := time.Parse(time.RFC3339, u.CreatedAt)
		if err != nil {
			createdAt = time.Now()
		}

		_, err = pgPool.Exec(ctx,
			`INSERT INTO users (id, email, password_hash, full_name, created_at)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (email) DO NOTHING`,
			u.ID, u.Email, string(hash), u.FullName, createdAt,
		)
		if err != nil {
			log.Printf("aviso: error insertando usuario %s: %v", u.Email, err)
			usersSkipped++
			continue
		}

		userExists[u.ID] = true
		usersCreated++
		if (i+1)%200 == 0 {
			log.Printf("progreso usuarios: %d/%d", i+1, len(data.Users))
		}
	}
	log.Printf("usuarios: %d creados, %d saltados", usersCreated, usersSkipped)

	// 2. Crear TODAS las cuentas de cada usuario (ya no solo la primera -
	// el esquema ahora soporta multiples cuentas por usuario).
	accountNumberToTBID := map[string]tbtypes.Uint128{}
	accountsCreated, accountsSkipped := 0, 0
	for i, acc := range data.Accounts {
		if !userExists[acc.UserID] {
			accountsSkipped++
			continue
		}

		tbID, err := tbClient.CreateUserAccount()
		if err != nil {
			log.Printf("aviso: error creando cuenta tigerbeetle %s: %v", acc.AccountNumber, err)
			accountsSkipped++
			continue
		}

		accountType := acc.AccountType
		if accountType == "" {
			accountType = "checking"
		}
		currency := acc.Currency
		if currency == "" {
			currency = "USD"
		}

		_, err = pgPool.Exec(ctx,
			`INSERT INTO accounts (user_id, tigerbeetle_account_id, account_number, account_type, currency)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (tigerbeetle_account_id) DO NOTHING`,
			acc.UserID, tbID.String(), acc.AccountNumber, accountType, currency,
		)
		if err != nil {
			log.Printf("aviso: error insertando cuenta %s: %v", acc.AccountNumber, err)
			accountsSkipped++
			continue
		}

		accountNumberToTBID[acc.AccountNumber] = tbID

		initial := uint64(math.Round(acc.InitialBalance))
		if initial > 0 {
			if err := tbClient.Deposit(tbID, initial); err != nil {
				log.Printf("aviso: error fondeando cuenta %s: %v", acc.AccountNumber, err)
			}
		}

		accountsCreated++
		if (i+1)%200 == 0 {
			log.Printf("progreso cuentas: %d/%d", i+1, len(data.Accounts))
		}
	}
	log.Printf("cuentas: %d creadas, %d saltadas", accountsCreated, accountsSkipped)

	// 3. Replay de transacciones, en el orden en que vienen en el archivo.
	txExecuted, txSkipped := 0, 0
	for i, tx := range data.Transactions {
		amount := uint64(math.Round(tx.Amount))
		if amount == 0 {
			txSkipped++
			continue
		}

		fromID, fromOK := accountNumberToTBID[tx.FromAccount]
		toID, toOK := accountNumberToTBID[tx.ToAccount]

		var opErr error
		switch {
		case tx.FromAccount == "EXTERNAL" && toOK:
			opErr = tbClient.Deposit(toID, amount)
		case fromOK && tx.ToAccount == "EXTERNAL":
			opErr = tbClient.Withdraw(fromID, amount)
		case fromOK && toOK:
			opErr = tbClient.Transfer(fromID, toID, amount)
		default:
			txSkipped++
			continue
		}

		if opErr != nil {
			txSkipped++
		} else {
			txExecuted++
		}

		if (i+1)%500 == 0 {
			log.Printf("progreso transacciones: %d/%d", i+1, len(data.Transactions))
		}
	}
	log.Printf("transacciones: %d ejecutadas, %d saltadas", txExecuted, txSkipped)
	log.Println("seed completo.")
}
