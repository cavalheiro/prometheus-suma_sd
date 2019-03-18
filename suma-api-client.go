package main

import (
  "github.com/kolo/xmlrpc"
)

type clientRef struct {
  Id            int        `xmlrpc:"id"`
}

type clientDetail struct {
  Id            int        `xmlrpc:"id"`
  Hostname      string     `xmlrpc:"hostname"`
  Entitlements  []string   `xmlrpc:"addon_entitlements"`
}

// Attempt to login in SUSE Manager Server and get an auth token
func Login(host string, user string, pass string) (string, error) {
  client, _ := xmlrpc.NewClient(host, nil)
  var result string
  err := client.Call("auth.login", []interface{}{user, pass}, &result)
  return result, err
}

// Logout from SUSE Manager API
func Logout(host string, token string) (error) {
  client, _ := xmlrpc.NewClient(host, nil)
  err := client.Call("auth.logout", token, nil)
  return err
}

// Get client list
func GetClientList(host string, token string) ([]clientRef, error) {
  client, _ := xmlrpc.NewClient(host, nil)
  var result []clientRef
  err := client.Call("system.listSystems", token, &result)
  return result, err
}

// Get client list
func GetClientDetails(host string, token string, systemId int) (clientDetail, error) {
  client, _ := xmlrpc.NewClient(host, nil)
  var result clientDetail
  err := client.Call("system.getDetails", []interface{}{token, systemId}, &result)
  return result, err
}
