// Package jbbsreader provides functionality for reading from a specified JBBS board and feeding
// updated responses to a chan.
package jbbsreader

import "fmt"

// Client is a client for reading from JBBS.
type Client struct {
}

// Read occasionaly reads from the JBBS and feed it to ch.
func (c *Client) Read(ch chan<- Res) {
}

// NewClient constructs a Client.
func NewClient() {
  fmt.Println("Hello")
}

// Thread represents each thread on JBBS.
type Thread struct {
}

// Res represents each response written to JBBS threads.
type Res struct {
  ParentThread *Thread
}

