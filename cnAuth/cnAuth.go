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

package cnAuth

import (
  "bufio"
  "bytes"
  "crypto/hmac"
  "crypto/sha256"
  "encoding/base64"
  "encoding/hex"
  "fmt"
  "github.com/pkg/errors"
  "os"
  "time"
)

type CnAuth struct {
  keys map[string]string
}

func NewCnAuthFromFile( file *os.File ) (*CnAuth, error) {
  cnAuth := new (CnAuth)
  err := cnAuth.parseConfigFile(file)
  if err != nil {
    return nil, err
  }
  return cnAuth, nil
}


/* legacy: parse strange key file format
kapi_id="001";kapi_key="a27f9e73fdde6a5005879c273c9aea5e8d917eec77bbdfd73272c0af9b4c6b7a";kapi_groups="watcher";eval ugroups_${kapi_id}=${kapi_groups};eval ukey_${kapi_id}=${kapi_key}
 */

func ( cnAuth *CnAuth ) parseConfigFile(file *os.File) error {
  cnAuth.keys = make( map[string]string, 0 )
  scanner := bufio.NewScanner(file)
  for scanner.Scan() {
    line := []byte(scanner.Text())
    fieldsKV :=bytes.Split( bytes.Trim(line, " "), []byte(";") )

    // only first 3 kv pairs are relevant
    var keyLabel string
    var keyHex string
    for fkv := 0; fkv<3; fkv++ {
      kv := bytes.Split( bytes.Trim(fieldsKV[fkv], " "), []byte("=") )

      switch string(kv[0]) {
      case "kapi_id":
        keyLabel = string(bytes.Trim(kv[1],"\""))
        break
      case "kapi_key":
        keyHex = string(bytes.Trim(kv[1],"\""))
        break
      }

    }
    if keyLabel != "" && keyHex != "" {
      cnAuth.keys[keyLabel] = keyHex
    }
  }
  return scanner.Err()
}

/*

#!/bin/bash

k="9cf15759eb77400f2d0d54d9e3a5822fc5b1f49817f0a65e930a1ed6bf3f8a00"
id="003"

h64=$(echo -n "{\"alg\":\"HS256\",\"typ\":\"JWT\"}" | base64)
p64=$(echo -n "{\"id\":\"$id\",\"exp\":$((`date +"%s"`+10))}" | base64)
s=$(echo -n "$h64.$p64" | openssl dgst -hmac "$k" -sha256 -r | cut -sd ' ' -f1)
token="$h64.$p64.$s"

echo h64=$h64
echo p64=$p64
echo k=$k
echo token=$token
echo ""
curl --cacert dist/gatekeeper/cert.pem -H "Authorization: Bearer $token" https://127.0.0.1/getnewaddress
echo ""


 */

func ( cnAuth *CnAuth ) BearerFromKey( keyLabel string ) (string, error) {
  if keyHex, ok := cnAuth.keys[keyLabel]; ok {
    header := "{\"alg\":\"HS256\",\"typ\":\"JWT\"}"
    payload := fmt.Sprintf("{\"id\":\"%s\",\"exp\":%d}", keyLabel, time.Now().Unix()+10 )

    h64 := base64.StdEncoding.EncodeToString( []byte(header) )
    p64 := base64.StdEncoding.EncodeToString( []byte(payload) )
    toSign := h64+"."+p64
    h := hmac.New( sha256.New, []byte(keyHex) )
    h.Write([]byte(toSign))
    sha := hex.EncodeToString(h.Sum(nil))
    return "Bearer "+toSign+"."+sha, nil
  }
  return "", errors.New("No such key with label "+keyLabel )
}
