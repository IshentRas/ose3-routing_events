package main

import (
     "bufio"
     "crypto/tls"
     "encoding/json"
     "fmt"
     "io/ioutil"
     "log"
     "net/http"
     "os"
     "strconv"
)

var secret_path = "/run/secrets/kubernetes.io/serviceaccount/token"
var master_uri  = os.Getenv("OPENSHIFT_MASTER")

func init_connection(path_connection , secret string) *http.Response {

     // Disable secure check on HTTPS connections
     tr := &http.Transport{
         TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
     }
     
     uri_to_connect := master_uri + path_connection

     client := &http.Client{Transport: tr}

     req, err := http.NewRequest("GET", uri_to_connect , nil)

     req.Header.Set("Authorization","Bearer " + string(secret))

     response, err := client.Do(req)

     if err != nil {
         fmt.Println(err)
     }

     return response

}

func main() {

     if len(master_uri) < 1 {
         log.Fatal("Could not read environment variable \"OPENSHIFT_MASTER\"")
     }

     secret_token, err := ioutil.ReadFile(secret_path)
     if err != nil {
        panic(err)
     }

     response_oapi := init_connection("/oapi/v1/routes?watch" , string(secret_token))

     reader := bufio.NewReader(response_oapi.Body)

     for {
        line, err := reader.ReadBytes('\n')
        if err != nil {
            fmt.Println(err)
        }
        // Decode the json object
        var u map[string]interface{}
        err = json.Unmarshal([]byte(line), &u)
        if err != nil {
            panic(err)
        }
        action := u["type"].(string)
        object := u["object"].(map[string]interface{})
        metadata := object["metadata"].(map[string]interface{})
        spec := object["spec"].(map[string]interface{})
        endpoint_name := spec["to"].(interface{}).(map[string]interface{})["name"]

        switch action {
             case "ADDED" : 
                  response_api := init_connection("/api/v1/namespaces/" + metadata["namespace"].(string) + "/endpoints/" + endpoint_name.(string) , string(secret_token))
                  body, err := ioutil.ReadAll(response_api.Body)
                  if err != nil {
                      panic(err)
                  }

                  var v map[string]interface{}
                  err = json.Unmarshal([]byte(body), &v)
                  if err != nil {
                      panic(err)
                  }

                  var pool_member []string
                  var size_endpoint = len(v["subsets"].([]interface{})[0].(map[string]interface{})["addresses"].([]interface{}))
                  
                  PORT := strconv.Itoa(int(v["subsets"].([]interface{})[0].(map[string]interface{})["ports"].([]interface{})[0].(map[string]interface{})["port"].(float64)))
 
                  for i := 0 ; i < size_endpoint ; i++ {
                      IP   := v["subsets"].([]interface{})[0].(map[string]interface{})["addresses"].([]interface{})[i].(map[string]interface{})["ip"]
                      pool_member = append(pool_member,IP.(string) + ":" + PORT)
                  } 

                  fmt.Printf("\nCREATE NEW ROUTE\n================\n")
                  fmt.Printf("APP = %v\n",metadata["name"])
                  fmt.Printf("NAMESPACE = %v\n",metadata["namespace"])
                  fmt.Printf("FQDN TO EXPOSE = %v\n",spec["host"])
                  fmt.Printf("NAME OF ENDPOINT = %v\n",endpoint_name)
                  fmt.Printf("SIZE POOL MEMBER = %v\n",size_endpoint)
                  fmt.Printf("POOL MEMBER = %v\n\n",pool_member)

             case "DELETED" :
                  response_api := init_connection("/api/v1/namespaces/" + metadata["namespace"].(string) + "/endpoints/" + endpoint_name.(string) , string(secret_token))
                  body, err := ioutil.ReadAll(response_api.Body)
                  if err != nil {
                      panic(err)
                  }
                  var v map[string]interface{}
                  err = json.Unmarshal([]byte(body), &v)
                  if err != nil {
                      panic(err)
                  }

                  fmt.Printf("\nDELETE EXISTING ROUTE\n=====================\n")
                  fmt.Printf("APP = %v\n",metadata["name"])
                  fmt.Printf("NAMESPACE = %v\n",metadata["namespace"])
        }
     } 
}
