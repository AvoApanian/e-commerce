package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
)

func DeleteAllUsers(db *pgx.Conn) {
	_, err := db.Exec(context.Background(), "DELETE FROM user_bdd")
	if err != nil {
		fmt.Printf("Erreur suppression: %v\n", err)
		return
	}
	fmt.Println("Base de données vidée")
}

func PrintAllUsers(db *pgx.Conn) {
	fmt.Println("\n=== UTILISATEURS ENREGISTRÉS ===")

	rows, err := db.Query(context.Background(), 
		"SELECT id, mail, password, theme, bankName, cartNum, mm_aa, cvv FROM user_bdd")
	if err != nil {
		fmt.Printf("Erreur lecture: %v\n", err)
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, mail, password string
		var theme *bool
		var bankName, cartNum, mm_aa, cvv *int32

		if err := rows.Scan(&id, &mail, &password, &theme, &bankName, &cartNum, &mm_aa, &cvv); err != nil {
			fmt.Printf("Erreur ligne %d: %v\n", count+1, err)
			continue
		}
		
	
		var themeStr string
		if theme != nil {
			themeStr = fmt.Sprintf("%v", *theme)
		} else {
			themeStr = "NULL"
		}

		var bankNameStr, cartNumStr, mmAaStr, cvvStr string
		if bankName != nil {
			bankNameStr = fmt.Sprintf("%d", *bankName)
		} else {
			bankNameStr = "NULL"
		}
		if cartNum != nil {
			cartNumStr = fmt.Sprintf("%d", *cartNum)
		} else {
			cartNumStr = "NULL"
		}
		if mm_aa != nil {
			mmAaStr = fmt.Sprintf("%d", *mm_aa)
		} else {
			mmAaStr = "NULL"
		}
		if cvv != nil {
			cvvStr = fmt.Sprintf("%d", *cvv)
		} else {
			cvvStr = "NULL"
		}

		fmt.Printf(
			"\n#%d\nID: %s\nEmail: %s\nPassword: %s\nTheme: %s\nBankName: %s\nCartNum: %s\nMM_AA: %s\nCVV: %s\n",
			count+1, id, mail, password, themeStr, bankNameStr, cartNumStr, mmAaStr, cvvStr,
		)
		count++
	}

	fmt.Printf("\nTotal: %d utilisateur(s)\n", count)
}

func Select(conn *pgx.Conn, table string, mail string) (string, error) {
	query := fmt.Sprintf("SELECT mail FROM %s WHERE mail = $1", table)

	var result string
	err := conn.QueryRow(context.Background(), query, mail).Scan(&result)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return "", nil
		}
		return "", err
	}

	return result, nil
}


func addMesh(
	bdd *pgx.Conn,
	name, description, image, mesh string,
	rating, price, count int32,
) {
	query := `
		INSERT INTO product (
			name, description, image, mesh, rating, price, count
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
	`

	_, err := bdd.Exec(context.Background(),
		query, name, description, image, mesh,
		rating, price, count,
	)

	if err != nil {
		fmt.Printf("Erreur d'ajout des données dans la table product : %v\n", err)
	}
}


func PrintAllProduct(bdd *pgx.Conn) {
	row, err := bdd.Query(context.Background(),
		"SELECT id, name, description, image, mesh, rating, price, count FROM product",
	)
	if err != nil {
		fmt.Printf("Erreur de print depuis la fonction PrintAllProduct : %v\n", err)
		return
	}
	defer row.Close() 

	for row.Next() {
		var id string
		var name, description, image, mesh string
		var rating, price, count int32

		err := row.Scan(&id, &name, &description, &image, &mesh, &rating, &price, &count)
		if err != nil {
			fmt.Printf("Erreur lors de la lecture d'une ligne : %v\n", err)
			continue
		}

		fmt.Printf("ID: %s\nNom: %s\nDescription: %s\nImage: %s\nMesh: %s\nNote: %d/5\nPrix: $%d\nStock: %d\n\n",
			id, name, description, image, mesh, rating, price, count)
	}

	if err := row.Err(); err != nil {
		fmt.Printf("Erreur lors du parcours des lignes : %v\n", err)
	}
}
func deleteProductByName(bdd *pgx.Conn, name string) {
	query := `
		DELETE FROM product
		WHERE name = $1
	`

	_, err := bdd.Exec(context.Background(), query, name)

	if err != nil {
		fmt.Printf("Erreur lors de la suppression du produit %s : %v\n", name, err)
		return
	}

	fmt.Printf("Produit %s supprimé avec succès.\n", name)
}


