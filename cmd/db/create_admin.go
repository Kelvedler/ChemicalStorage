package main

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/term"

	"github.com/Kelvedler/ChemicalStorage/pkg/common"
	"github.com/Kelvedler/ChemicalStorage/pkg/db"
	"github.com/Kelvedler/ChemicalStorage/pkg/env"
)

func getPassword() (password []byte) {
	for {
		fmt.Println("Enter password: ")
		password1, err := term.ReadPassword(0)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("Repeat password: ")
		password2, err := term.ReadPassword(0)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if string(password1) != string(password2) {
			fmt.Println("Passwords don't match")
		} else {
			return password1
		}
	}
}

func main() {
	password := getPassword()
	hashedPassword, err := common.HashPassword(password)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	adminUser := db.StorageUser{
		Name:     "Адміністратор",
		Password: hashedPassword,
		Role:     db.Admin,
	}
	ctx := context.Background()
	env.InitEnv()
	mainLogger := common.MainLogger()
	dbpool := db.GetConnectionPool(ctx, mainLogger)
	_, err = db.StorageUserCreate(ctx, dbpool, adminUser)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Admin created successfuly")
	}
	os.Exit(0)
}
