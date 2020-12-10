package main

import (
	"context"

	qmgoadapter "github.com/beiyanpiki/casbin-qmgo-adapter"
	"github.com/casbin/casbin/v2"
	"github.com/qiniu/qmgo"
)

func main() {
	// Initialize Qmgo.Client, Qmgo.Database and Qmgo.Collection.
	c, err := qmgo.NewClient(context.Background(), &qmgo.Config{Uri: "mongodb://127.0.0.1:27017"})
	if err != nil {
		panic(err)
	}
	coll := c.Database("Casbin").Collection("Casbin")
	// Initialize a Qmgo adapter and use it in a Casbin enforcer.
	a := qmgoadapter.NewAdapter(coll)

	e, err := casbin.NewEnforcer("examples/rbac_model.conf", a)
	if err != nil {
		panic(err)
	}

	// Load the policy from DB.
	e.LoadPolicy()

	// Check the permission.
	e.Enforce("alice", "data1", "read")

	// Modify the policy.
	// e.AddPolicy(...)
	// e.RemovePolicy(...)

	// Save the policy back to DB.
	e.SavePolicy()
}
