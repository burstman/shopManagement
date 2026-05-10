package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"shopDashboard/app/db"
	"shopDashboard/app/views/auth"
	"shopDashboard/app/views/layouts"

	"github.com/anthdm/superkit/kit"
	"golang.org/x/crypto/bcrypt"
)

func generatePassword() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

type AdminAuth struct {
	AdminID int
	Name    string
}

func (a AdminAuth) Check() bool { return a.AdminID > 0 }

func LoadAuth(k *kit.Kit) (kit.Auth, error) {
	sess := k.GetSession("admin")
	id, ok := sess.Values["admin_id"].(int)
	if !ok {
		return kit.DefaultAuth{}, nil
	}
	name, _ := sess.Values["admin_name"].(string)
	return AdminAuth{AdminID: id, Name: name}, nil
}

func HandleSetup(k *kit.Kit) error {
	has, err := db.HasAnySuperAdmin()
	if err != nil {
		return err
	}
	if has {
		http.Redirect(k.Response, k.Request, "/login", http.StatusSeeOther)
		return nil
	}

	if k.Request.Method == "POST" {
		name := k.Request.FormValue("name")
		email := k.Request.FormValue("email")

		if name == "" || email == "" {
			return k.Render(layouts.Base("Setup", auth.SetupPage("", "", "All fields are required")))
		}

		password := generatePassword()
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		admin, err := db.CreateSuperAdmin(name, email, string(hash))
		if err != nil {
			return k.Render(layouts.Base("Setup", auth.SetupPage(name, email, "Failed to create admin: "+err.Error())))
		}

		return k.Render(layouts.Base("Setup", auth.SetupSuccess(admin.Name, admin.Email, password)))
	}

	return k.Render(layouts.Base("Setup", auth.SetupPage("", "", "")))
}

func HandleLogin(k *kit.Kit) error {
	if k.Request.Method == "POST" {
		email := k.Request.FormValue("email")
		password := k.Request.FormValue("password")

		admin, err := db.GetSuperAdminByEmail(email)
		if err != nil {
			return k.Render(layouts.Base("Login", auth.LoginPage(email, "Invalid email or password")))
		}

		if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
			return k.Render(layouts.Base("Login", auth.LoginPage(email, "Invalid email or password")))
		}

		sess := k.GetSession("admin")
		sess.Values["admin_id"] = admin.ID
		sess.Values["admin_name"] = admin.Name
		sess.Save(k.Request, k.Response)

		http.Redirect(k.Response, k.Request, "/", http.StatusSeeOther)
		return nil
	}

	return k.Render(layouts.Base("Login", auth.LoginPage("", "")))
}

func HandleLogout(k *kit.Kit) error {
	sess := k.GetSession("admin")
	sess.Values["admin_id"] = 0
	sess.Values["admin_name"] = ""
	sess.Save(k.Request, k.Response)
	http.Redirect(k.Response, k.Request, "/login", http.StatusSeeOther)
	return nil
}

func HandleAdmins(k *kit.Kit) error {
	if k.Request.Method == "POST" {
		name := k.Request.FormValue("name")
		email := k.Request.FormValue("email")

		if name == "" || email == "" {
			admins, _ := db.GetAllSuperAdmins()
			return k.Render(layouts.Base("Admins", auth.AdminsPage(admins, "All fields are required")))
		}

		password := generatePassword()
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		admin, err := db.CreateSuperAdmin(name, email, string(hash))
		if err != nil {
			admins, _ := db.GetAllSuperAdmins()
			return k.Render(layouts.Base("Admins", auth.AdminsPage(admins, "Failed to create admin: "+err.Error())))
		}

		return k.Render(layouts.Base("Admins", auth.AdminCreated(admin.Name, admin.Email, password)))
	}

	admins, err := db.GetAllSuperAdmins()
	if err != nil {
		return err
	}
	return k.Render(layouts.Base("Admins", auth.AdminsPage(admins, "")))
}
