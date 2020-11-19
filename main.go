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
  "cyphernode_welcome/cnAuth"
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
  "path/filepath"
  "reflect"
  "strings"
)

type BlockChainInfo struct {
  Verificationprogress float32  `json:"verificationprogress"`
  InitialBlockdownload bool  `json:"initialblockdownload"`
}

type TemplateData struct {
  InstallationInfo *InstallationInfo
  ForwardedPrefix string
  FeatureByName map[string]bool
}

type FeatureState struct {
  Name string `json:"name"`
  Working bool `json:"working"`
}

type Feature struct {
  Name string `json:"name"`
  Label string `json:"label"`
  Host string `json:"host"`
  Networks []string `json:"networks"`
  Docker string `json:"docker"`
  Extra interface{} `json:"extra"`
  Error bool
}

type OptionalFeature struct {
  Feature
  Active bool `json:"active"`
}

type InstallationState struct {
  FeatureStates []FeatureState `json:"features"`
}

type InstallationInfo struct {
  ApiVersions []string `json:"api_versions"`
  SetupVersion string `json:"setup_version"`
  BitcoinVersion string `json:"bitcoin_version"`
  Features []Feature `json:"features"`
  OptionalFeatures []OptionalFeature `json:"optional_features"`
  DevMode bool `json:"devmode"`
}

var auth *cnAuth.CnAuth
var statsKeyLabel string
var rootTemplate *template.Template
var statusEndpoint string
var installationInfoEndpoint string
var installationStateEndpoint string
var configArchiveEndpoint string
var certsEndpoint string
var passwordHashes map[string][]byte

var httpClient *http.Client
var log = logging.MustGetLogger("main")


func RootHandler(w http.ResponseWriter, req *http.Request) {
  installationInfo, err := getInstallatioInfo()
  if err != nil {
    log.Errorf("Error retrieving installation info %s", err )
  }
  installationInfo.ForwardedPrefix = req.Header.Get("X-Forwarded-Prefix")
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

func  getInstallatioInfo() (*TemplateData,error) {
  log.Info("getInstallationState")
  body,err := getBodyUsingAuth(os.Getenv("CYPHERNODE_URL")+installationStateEndpoint)

  if err != nil {
    log.Errorf("getInstallationState: %s", err)
    return nil,err
  }

  log.Infof("getInstallationState: %", string(body))


  installationState := new(InstallationState)

  err = json.Unmarshal( body, &installationState)

  if err != nil {
    log.Errorf("getInstallationState: %s", err)
    return nil,err
  }

  log.Info("getInstallationState: json done")


  log.Info("getInstallationInfo")
  body,err = getBodyUsingAuth(os.Getenv("CYPHERNODE_URL")+installationInfoEndpoint)

  if err != nil {
    log.Errorf("getInstallationInfo: %s", err)
    return nil,err
  }

  log.Infof("getInstallationInfo: %", string(body))


  installationInfo := new(InstallationInfo)

  err = json.Unmarshal( body, &installationInfo)

  if err != nil {
    log.Errorf("getInstallationInfo: %s", err)
    return nil,err
  }

  log.Info("getInstallationInfo: json done")
  templateData := new(TemplateData)

  templateData.FeatureByName = make( map[string]bool )

  for j:=0; j< len(installationInfo.Features); j++ {
    templateData.FeatureByName[installationInfo.Features[j].Label] = true
  }
  for j:=0; j< len(installationInfo.OptionalFeatures); j++ {
    templateData.FeatureByName[installationInfo.OptionalFeatures[j].Label] = installationInfo.OptionalFeatures[j].Active
  }

  for i:=0; i< len(installationState.FeatureStates); i++ {
    found := false
    for j:=0; j< len(installationInfo.Features); j++ {
      if installationInfo.Features[j].Label == installationState.FeatureStates[i].Name {
        installationInfo.Features[j].Error = !installationState.FeatureStates[i].Working
        found = true
        break
      }
    }
    for j:=0; !found && j< len(installationInfo.OptionalFeatures); j++ {
      if installationInfo.OptionalFeatures[j].Label == installationState.FeatureStates[i].Name {
        installationInfo.OptionalFeatures[j].Error = !installationState.FeatureStates[i].Working
        break
      }
    }
  }

  templateData.InstallationInfo = installationInfo

  return templateData,nil
}

func VerificationProgressHandler(w http.ResponseWriter, r *http.Request) {

  body,err := getBodyUsingAuth(os.Getenv("CYPHERNODE_URL")+statusEndpoint)

  if err != nil {
    log.Errorf("VerificationProgressHandler: %s", err)
    w.WriteHeader(503 )
    return
  }

  blockChainInfo := new( BlockChainInfo )

  err = json.Unmarshal( body, &blockChainInfo )

  if err != nil {
    log.Errorf("VerificationProgressHandler: %s", err)
    w.WriteHeader(503 )
    return
  }

  w.Header().Set("Content-Type", "application/json")
  result, err := json.Marshal(&blockChainInfo)
  fmt.Fprint(w, bytes.NewBuffer(result))
}


func ConfigHandler(w http.ResponseWriter, r *http.Request) {

  body,err := getBodyUsingAuth(os.Getenv("CYPHERNODE_URL")+configArchiveEndpoint)

  if err != nil {
    log.Errorf("ConfigHandler: %s", err)
    w.WriteHeader(503 )
    return
  }

  w.Header().Set("Content-Type", "application/x-7z-compressed")
  fmt.Fprint(w, bytes.NewBuffer(body))
}

func CertsHandler(w http.ResponseWriter, r *http.Request) {

  body,err := getBodyUsingAuth(os.Getenv("CYPHERNODE_URL")+certsEndpoint)

  if err != nil {
    log.Errorf("CertsHandler: %s", err)
    w.WriteHeader(503 )
    return
  }

  w.Header().Set("Content-Type", "application/x-7z-compressed")
  fmt.Fprint(w, bytes.NewBuffer(body))
}

func Secret(user, realm string) string {
  if user == "john" {
    // password is "hello"
    return "$1$dlPL2MqE$oQmn16q49SqdmhenQuNgs1"
  }
  return ""
}

func main() {

  viper.SetConfigName("config")
  viper.AddConfigPath("/data")
  viper.AddConfigPath("data")

  err := viper.ReadInConfig()

  if err != nil {
    log.Errorf("Error loading config.toml: %s", err)
    return
  }

  keysFilePath := viper.GetString("security.key_file")
  statsKeyLabel = viper.GetString("security.key_label")
  certFile := viper.GetString("security.cert_file")
  statusEndpoint = viper.GetString("api.status_endpoint")
  installationInfoEndpoint = viper.GetString("api.installation_info_endpoint")
  installationStateEndpoint = viper.GetString("api.installation_state_endpoint")
  configArchiveEndpoint = viper.GetString("api.config_archive_endpoint")
  certsEndpoint = viper.GetString("api.certs_endpoint")
  listenTo := viper.GetString("server.listen")
  indexTemplate := viper.GetString("server.index_template")

  funcMap := template.FuncMap {
    "toString": func( v interface{} ) string {
      v2 := reflect.ValueOf(v)
      if v2.Kind() == reflect.Slice {
        s := make( []string, v2.Len() )
        for i := 0; i < v2.Len(); i++ {
          s[i]=v2.Index(i).Interface().(string)
        }
        return strings.Join( s, ", ")
      }
      return fmt.Sprintf("%v", v)
    },
    "toTitle": strings.Title,
  }

  rootTemplate, err = template.New(filepath.Base(indexTemplate)).Funcs(funcMap).ParseFiles(indexTemplate)

  if err != nil {
    log.Errorf("Error loading root template: %s", err)
    log.Error(err)
    return
  }

  caCert, err := ioutil.ReadFile(certFile)
  if err != nil {
    log.Errorf("Error loading cert: %s", err)
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
    log.Errorf("Error loading keys file: %s", err)
    log.Error(err)
    return
  }

  auth, err = cnAuth.NewCnAuthFromFile( file )
  file.Close()

  if err != nil {
    log.Errorf("Error creating auther: %s", err)
    log.Error(err)
    return
  }

  log.Infof("Started cyphernode status page backend. URL Port [%v] ",listenTo)
  log.Infof( "Using %s as API base url.", os.Getenv("CYPHERNODE_URL") )

  router := mux.NewRouter()

  router.HandleFunc("/", RootHandler)
  router.HandleFunc("/verificationprogress", VerificationProgressHandler)
  router.HandleFunc("/config.7z", ConfigHandler)
  router.HandleFunc("/certs.7z", CertsHandler)

  router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

  http.Handle("/", router)
  route := router.PathPrefix("/static")

  log.Fatal(route, http.ListenAndServe(listenTo, nil))
}
