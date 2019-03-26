/*
 * MIT License
 *
 * Copyright (c) 2019 schulterklopfer/SKP
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILIT * Y, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package main

import (
  "bytes"
  "crypto/tls"
  "crypto/x509"
  "cyphernode_status/cnAuth"
  "encoding/json"
  "fmt"
  "github.com/gorilla/mux"
  "github.com/op/go-logging"
  "github.com/pkg/errors"
  "github.com/spf13/viper"
  "html/template"
  "io/ioutil"
  "net/http"
  "os"
)

type BlockChainInfo struct {
  Verificationprogress float32  `json:"verificationprogress"`
}

type InstallationInfoFeature struct {
  Name string `json:"name"`
  Working bool `json:"working"`
}

type InstallationInfoContainer struct {
  Name string `json:"name"`
  Active bool `json:"active"`
}

type InstallationInfo struct {
  Features []InstallationInfoFeature  `json:"features"`
  Containers []InstallationInfoContainer  `json:"containers"`
}



var auth *cnAuth.CnAuth
var statsKeyLabel string
var rootTemplate *template.Template
var statusUrl string
var installationInfoUrl string
var configArchiveUrl string
var certsUrl string

var httpClient *http.Client
var log = logging.MustGetLogger("main")

func RootHandler(w http.ResponseWriter, _ *http.Request) {
  installationInfo, err := getInstallatioInfo()
  if err != nil {
    log.Errorf("Error retrieving installation info %s", err )
  }
  rootTemplate.Execute(w, installationInfo)
}

func getBodyUsingAuth( url string ) ([]byte,error) {

  req, err := http.NewRequest("GET", url, nil)
  if err != nil {
    return nil, err
  }

  bearer, err := auth.BearerFromKey(statsKeyLabel)
  if err != nil {
    return nil, err
  }

  req.Header.Set("Authorization", bearer )
  res,err := httpClient.Do(req)
  if err != nil {
    return nil, err
  }

  defer res.Body.Close()

  if res.StatusCode == 0 {
    return nil, err
  }

  if res.StatusCode != 200 {
    return nil, errors.New("Unexpected http status code")
  }

  body, err := ioutil.ReadAll(res.Body)

  if res.StatusCode == 0 {
    return nil, err
  }

  return body, nil
}

func  getInstallatioInfo() (*InstallationInfo,error) {

  body,err := getBodyUsingAuth( installationInfoUrl )

  installationInfo := new( InstallationInfo )

  err = json.Unmarshal( body, &installationInfo )

  fmt.Println( string(body))

  if err != nil {
    return nil,err
  }

  return installationInfo,nil
}

func VerificationProgressHandler(w http.ResponseWriter, r *http.Request) {

  body,err := getBodyUsingAuth( statusUrl )

  blockChainInfo := new( BlockChainInfo )

  err = json.Unmarshal( body, &blockChainInfo )

  if err != nil {
    w.WriteHeader(503 )
    return
  }

  w.Header().Set("Content-Type", "application/json")
  result, err := json.Marshal(&blockChainInfo)
  fmt.Fprint(w, bytes.NewBuffer(result))
}


func ConfigHandler(w http.ResponseWriter, r *http.Request) {

  body,err := getBodyUsingAuth( configArchiveUrl )

  if err != nil {
    w.WriteHeader(503 )
    return
  }

  w.Header().Set("Content-Type", "application/x-7z-compressed")
  fmt.Fprint(w, bytes.NewBuffer(body))
}

func CertsHandler(w http.ResponseWriter, r *http.Request) {

  body,err := getBodyUsingAuth( certsUrl )

  if err != nil {
    w.WriteHeader(503 )
    return
  }

  w.Header().Set("Content-Type", "application/x-7z-compressed")
  fmt.Fprint(w, bytes.NewBuffer(body))
}


func main() {

  viper.SetConfigName("config")
  viper.AddConfigPath("data")

  err := viper.ReadInConfig()

  if err != nil {
    log.Error(err)
    return
  }

  keysFilePath := viper.GetString("gatekeeper.key_file")
  statsKeyLabel = viper.GetString("gatekeeper.key_label")
  statusUrl = viper.GetString("gatekeeper.status_url")
  installationInfoUrl = viper.GetString("gatekeeper.installation_info_url")
  configArchiveUrl = viper.GetString("gatekeeper.config_archive_url")
  certsUrl = viper.GetString("gatekeeper.certs_url")
  certFile := viper.GetString("gatekeeper.cert_file")
  listenTo := viper.GetString("server.listen")
  indexTemplate := viper.GetString("server.index_template")
  rootTemplate, err = template.ParseFiles(indexTemplate)

  if err != nil {
    log.Error(err)
    return
  }

  caCert, err := ioutil.ReadFile(certFile)
  if err != nil {
    log.Error(err)
    return
  }

  caCertPool := x509.NewCertPool()
  caCertPool.AppendCertsFromPEM(caCert)

  httpClient = &http.Client{
    Transport: &http.Transport{
      TLSClientConfig: &tls.Config{
        RootCAs: caCertPool,
      },
    },
  }


  file, err := os.Open(keysFilePath)

  if err != nil {
    log.Error(err)
    return
  }

  auth, err = cnAuth.NewCnAuthFromFile( file )
  file.Close()

  if err != nil {
    log.Error(err)
    return
  }

  log.Infof("Started cyphernode status page backend. URL Port [%v] ",listenTo)

  router := mux.NewRouter()
  router.HandleFunc("/", RootHandler)
  router.HandleFunc("/verificationprogress", VerificationProgressHandler)
  router.HandleFunc("/config.7z", ConfigHandler)
  router.HandleFunc("/certs.7z", CertsHandler)

  router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

  http.Handle("/", router)

  log.Fatal(http.ListenAndServe(listenTo, nil))
}
