package main

import (
	"bytes"
	_ "crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/inconshreveable/go-update"
)

const Version = "v1"

var (
	pageTemplate = `
		<!DOCTYPE html>
		<html>
		<head>
		<title> Server {{.Version}} </title>
		</head>
		<body>
		<h1>This server is version {{.Version}}</h1>
		<a href="check">Check for new version</a>
		<br>
		{{if .NewVersion}}New version is available: {{.NewVersion}} | <a
		href="install">Upgrade</a>{{end}}
		</body>
		</html>
		`
	ecdsaPublicKey = `
-----BEGIN PUBLIC KEY-----
MHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEL8ThbSyEucsCxnd4dCZR2hIy5nea54ko
O+jUUfIjkvwhCWzASm0lpCVdVpXKZXIe+NZ+44RQRv3+OqJkCCGzUgJkPNI3lxdG
9zu8rbrnxISV06VQ8No7Ei9wiTpqmTBB
-----END PUBLIC KEY-----
`
	strSig = `3066023100c3c7b5c39fcf8d6a7b56ed915c8b0bef10b7b4f420c728fcdc2f69558d88a518778c1fe7447ff9d8b15d3aefe3ee5307023100818f717ed9bdc1383edbb2c6c9c352ff713a8bea41f98e562db647f6b77118b992a1fdd2c154c7a86d7afd747c6fbc2a`
)

func main() {
	page, err := template.New("page").Parse(pageTemplate)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(Version)
	Status := Status{Version: Version, NewVersion: "v2"}

	updater := SecureUpdater{publicKey: ecdsaPublicKey, signature: strSig, binLoad: BinLoader{filePath: "v2.exe"}}
	ctx := Context{page: page, status: Status, binUpdate: updater}
	mux := http.NewServeMux()
	mux.Handle("/check", http.HandlerFunc(ctx.check))
	mux.Handle("/install", http.HandlerFunc(ctx.install))
	log.Fatal(http.ListenAndServe("localhost:8080", mux))
}

type Status struct {
	Version    string
	NewVersion string
}

type Context struct {
	page      *template.Template
	status    Status
	binUpdate Updater
}

type Loader interface {
	read() ([]byte, error)
}

type BinLoader struct {
	filePath string
}

func (bl BinLoader) read() ([]byte, error) {
	b, err := ioutil.ReadFile(bl.filePath)
	if err != nil {
		return []byte{}, err
	}
	return b, err
}

type Updater interface {
	update() error
}

type SecureUpdater struct {
	publicKey string
	signature string
	binLoad   Loader
}

func (updater SecureUpdater) update() error {
	newFile, err := updater.binLoad.read()
	if err != nil {
		return err
	}
	signature, err := hex.DecodeString(updater.signature)
	if err != nil {
		return err
	}
	opts := update.Options{Signature: signature}
	err = opts.SetPublicKeyPEM([]byte(updater.publicKey))
	if err != nil {
		return err
	}
	err = update.Apply(bytes.NewReader(newFile), opts)
	if err != nil {
		return err
	}
	return err
}

func (ctx Context) install(w http.ResponseWriter, req *http.Request) {
	err := ctx.binUpdate.update()
	if err != nil {
		sendJson("update failed!", w)
		return
	}
	var latest string = "Updated SUCCESS to version: " + ctx.status.NewVersion
	sendJson(latest, w)
}

func (ctx Context) check(w http.ResponseWriter, req *http.Request) {
	if err := ctx.page.Execute(w, ctx.status); err != nil {
		log.Fatal(err)
	}
}

func sendJson(payload interface{}, w http.ResponseWriter) {
	jData, err := json.Marshal(payload)
	if err != nil {
		log.Fatal("Cannot marshal the data ", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jData)
}
