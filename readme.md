Casbin Qmgo Adapter 
====

Casbin Qmgo Adapter is the [Qmgo](https://github.com/qiniu/qmgo) adapter for [Casbin](https://github.com/casbin/casbin). With this library, Casbin can load policy from MongoDB or save policy to it.

## Installation

    go get -u github.com/beiyanpiki/casbin-qmgo-adapter

## Simple Example

```go
package main

import (
	"context"

	"casbinqmgoadapter"

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
	a := casbinqmgoadapter.NewAdapter(coll)

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
```

## Filtered Policies

```go
import "go.mongodb.org/mongo-driver/bson"

// This adapter also implements the FilteredAdapter interface. This allows for
// efficent, scalable enforcement of very large policies:
filter := &bson.M{"v0": "alice"}
e.LoadFilteredPolicy(filter)

// The loaded policy is now a subset of the policy in storage, containing only
// the policy lines that match the provided filter. This filter should be a
// valid MongoDB selector using BSON. A filtered policy cannot be saved.
```

## Getting Help

- [Casbin](https://github.com/casbin/casbin)

## License

This project is under Apache 2.0 License. See the [LICENSE](LICENSE) file for the full license text.
