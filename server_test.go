package main

import (
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	_ "reflect"
	"strings"
	"testing"
)

var (
	pageTemplateTest = `
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

	testEcdsaPub = `
-----BEGIN PUBLIC KEY-----
MHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEL8ThbSyEucsCxnd4dCZR2hIy5nea54ko
O+jUUfIjkvwhCWzASm0lpCVdVpXKZXIe+NZ+44RQRv3+OqJkCCGzUgJkPNI3lxdG
9zu8rbrnxISV06VQ8No7Ei9wiTpqmTBB
-----END PUBLIC KEY-----
`

	// signatureV2    = `3066023100c3c7b5c39fcf8d6a7b56ed915c8b0bef10b7b4f420c728fcdc2f69558d88a518778c1fe7447ff9d8b15d3aefe3ee5307023100818f717ed9bdc1383edbb2c6c9c352ff713a8bea41f98e562db647f6b77118b992a1fdd2c154c7a86d7afd747c6fbc2a`
	signatureBytes = `3065023100efb5177df440d95b97aba930eab9f3ccc40a7b1839283e9ceae5ff5a00f2a7d8d885744ea588c77255736f3a6d3fcb2d02300fac836e12cfe7018da7561e3c35b363f5efe4c689e1790f3d626b7fd86837e69ea900f2bd80c96e01e2da8a2cdb8748`

	fileBytes = []byte{0x01, 0x02, 0x03}
)

type FakeLoader struct{}

func (FakeLoader) read() ([]byte, error) {
	return fileBytes, nil
}

type FakeUpdater struct {
	err error
}

func (fu FakeUpdater) update() error {
	return fu.err
}

type MyErr struct {
	err string
}

func (e MyErr) Error() string {
	return e.err
}

func Test_SecureUpdater_CorrectUpdate1(t *testing.T) {
	updater := SecureUpdater{publicKey: ecdsaPublicKey, signature: signatureBytes, binLoad: FakeLoader{}}
	err := updater.update()
	if err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
}

func Test_SecureUpdater_IncorrectSignature(t *testing.T) {
	updater := SecureUpdater{publicKey: ecdsaPublicKey, signature: "", binLoad: FakeLoader{}}
	err := updater.update()
	if err == nil {
		t.Errorf("expected invalid signature, got: %v", err)
	}
}

func Test_SecureUpdater_IncorrectPublicKey(t *testing.T) {
	updater := SecureUpdater{publicKey: "", signature: strSig, binLoad: FakeLoader{}}
	err := updater.update()
	if err == nil {
		t.Errorf("expected couldn't parse PEM data, got: %v", err)
	}
}

func Test_BinLoader_CorrectFileRead(t *testing.T) {
	loader := BinLoader{"test.txt"}
	_, err := loader.read()
	if err != nil {
		t.Errorf("expected bytes, got: %v", err)
	}
}

func Test_BinLoader_CorrectContentRead(t *testing.T) {
	loader := BinLoader{"test.txt"}
	readBytes, _ := loader.read()
	current := string(readBytes[:])
	expected := "something to read"
	if current != expected {
		t.Errorf("expected \"something to read\", got: %v", current)
	}
}

func Test_Check(t *testing.T) {
	page, err := template.New("page").Parse(pageTemplateTest)
	if err != nil {
		log.Fatal(err)
	}

	ctx := Context{page: page, status: Status{"v1", "v2"}, binUpdate: FakeUpdater{nil}}
	res, err := runUrl(ctx.check, "/check")
	if err != nil {
		t.Errorf("Expected nil, received %s", err.Error())
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected %d, received %d", http.StatusOK, res.StatusCode)
	}
	_, err = ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
}

func Test_InstallUpdateSuccess(t *testing.T) {
	page, err := template.New("page").Parse(pageTemplateTest)
	if err != nil {
		log.Fatal(err)
	}
	ctx := Context{page: page, status: Status{"v1", "v2"}, binUpdate: FakeUpdater{nil}}
	res, err := runUrl(ctx.install, "/install")
	if err != nil {
		t.Errorf("Expected nil, received %s", err.Error())
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected %d, received %d", http.StatusOK, res.StatusCode)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	actual := string(body[:])
	expected := `"Updated SUCCESS to version: v2"`
	if actual != expected {
		t.Errorf("expected %v got %v", expected, actual)
	}
}

func Test_InstallUpdateFailed(t *testing.T) {
	page, err := template.New("page").Parse(pageTemplateTest)
	if err != nil {
		log.Fatal(err)
	}
	ctx := Context{page: page, status: Status{"v1", "v2"}, binUpdate: FakeUpdater{MyErr{"Update Failed"}}}
	res, err := runUrl(ctx.install, "/install")
	if err != nil {
		t.Errorf("Expected nil, received %s", err.Error())
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected %d, received %d", http.StatusOK, res.StatusCode)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	actual := string(body[:])
	expected := `"update failed!"`
	if actual != expected {
		t.Errorf("expected %v got %v", expected, actual)
	}
}

func runUrl(handler http.HandlerFunc, handleName string, params ...string) (*http.Response, error) {
	mux := http.NewServeMux()
	mux.Handle(handleName, http.HandlerFunc(handler))
	ts := httptest.NewServer(mux)
	defer ts.Close()
	allParams := strings.Join(params, "")
	res, err := http.Get(ts.URL + handleName + allParams)
	return res, err
}

//Just add this to be sure that file v2.exe is not broken for manula testing
// func Test_SecureUpdater_CorrectUpdate_v2(t *testing.T) {
// 	updater := SecureUpdater{publicKey: ecdsaPublicKey, signature: signatureV2, binLoad: BinLoader{filePath: "v2.exe"}}
// 	err := updater.update()
// 	if err != nil {
// 		t.Errorf("expected nil, got: %v", err)
// 	}
// }
