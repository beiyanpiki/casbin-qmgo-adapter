package qmgoadapter

import (
	"context"
	"errors"
	"strings"

	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"github.com/qiniu/qmgo"
	"github.com/qiniu/qmgo/options"
	"go.mongodb.org/mongo-driver/bson"
)

// CasbinRule represents a rule in Casbin.
type CasbinRule struct {
	PType string
	V0    string
	V1    string
	V2    string
	V3    string
	V4    string
	V5    string
}

type adapter struct {
	collection *qmgo.Collection
	filtered   bool
}

// NewAdapter is the constructor for Adapter. You need to provide a qmgo.Collection which is already defined.
// Example:
//
// 		client, _ := qmgo.NewClient(context.Background(), &qmgo.Config{
// 			Uri:              	"mongodb://127.0.0.1:27017/?authSource=admin&replicaSet=rs0",
// 			ConnectTimeoutMS:=	time.Second * 30,
// 			MaxPoolSize:      	100,
// 			Auth: 				qmgo.Credential{
// 				AuthSource: "admin",
// 				Username:   "root",
// 				Password:   "rootroot",
// 			},
// 		})
// 		coll := c.Database("Dbname").Collection("CasbinName")
// 		a, err := casbinqmgoadapter.NewAdapter(coll)
//
func NewAdapter(coll *qmgo.Collection) persist.Adapter {
	a := &adapter{
		collection: coll,
		filtered:   false,
	}
	coll.CreateOneIndex(
		context.Background(),
		options.IndexModel{Key: []string{
			"ptype",
			"v0",
			"v1",
			"v2",
			"v3",
			"v4",
			"v5",
		}, Unique: true},
	)
	return a
}

// NewFilteredAdapter is the constructor for FilteredAdapter.
// Casbin will not automatically call LoadPolicy() for a filtered adapter.
func NewFilteredAdapter(coll *qmgo.Collection) (persist.FilteredAdapter, error) {
	a := NewAdapter(coll)
	a.(*adapter).filtered = true

	return a.(*adapter), nil
}

// LoadPolicy loads policy from database.
func (a *adapter) LoadPolicy(model model.Model) error {
	return a.LoadFilteredPolicy(model, nil)
}

// IsFiltered returns true if the loaded policy has been filtered.
func (a *adapter) IsFiltered() bool {
	return a.filtered
}

// LoadFilteredPolicy loads matching policy lines from database.
// If not nil, the filter must be a valid MongoDB selector.
func (a *adapter) LoadFilteredPolicy(model model.Model, filter interface{}) error {
	if filter == nil {
		a.filtered = false
		filter = bson.D{{}}
	} else {
		a.filtered = true
	}

	lines := []CasbinRule{}

	a.collection.Find(
		context.Background(),
		filter,
	).All(&lines)

	for _, v := range lines {
		loadPolicyLine(v, model)
	}
	return nil
}

func loadPolicyLine(line CasbinRule, model model.Model) {
	var p = []string{line.PType,
		line.V0, line.V1, line.V2, line.V3, line.V4, line.V5}
	var lineText string
	if line.V5 != "" {
		lineText = strings.Join(p, ", ")
	} else if line.V4 != "" {
		lineText = strings.Join(p[:6], ", ")
	} else if line.V3 != "" {
		lineText = strings.Join(p[:5], ", ")
	} else if line.V2 != "" {
		lineText = strings.Join(p[:4], ", ")
	} else if line.V1 != "" {
		lineText = strings.Join(p[:3], ", ")
	} else if line.V0 != "" {
		lineText = strings.Join(p[:2], ", ")
	}

	persist.LoadPolicyLine(lineText, model)
}

// SavePolicy saves policy to database.
func (a *adapter) SavePolicy(model model.Model) error {
	if a.filtered {
		return errors.New("cannot save a filtered policy")
	}
	if err := a.collection.DropCollection(context.Background()); err != nil {
		return err
	}

	var lines []interface{}

	for ptype, ast := range model["p"] {
		for _, rule := range ast.Policy {
			line := savePolicyLine(ptype, rule)
			lines = append(lines, &line)
		}
	}

	for ptype, ast := range model["g"] {
		for _, rule := range ast.Policy {
			line := savePolicyLine(ptype, rule)
			lines = append(lines, &line)
		}
	}

	if _, err := a.collection.InsertMany(context.Background(), lines); err != nil {
		return err
	}

	return nil
}

func savePolicyLine(ptype string, rule []string) CasbinRule {
	line := CasbinRule{
		PType: ptype,
	}

	if len(rule) > 0 {
		line.V0 = rule[0]
	}
	if len(rule) > 1 {
		line.V1 = rule[1]
	}
	if len(rule) > 2 {
		line.V2 = rule[2]
	}
	if len(rule) > 3 {
		line.V3 = rule[3]
	}
	if len(rule) > 4 {
		line.V4 = rule[4]
	}
	if len(rule) > 5 {
		line.V5 = rule[5]
	}

	return line
}

// AddPolicy adds a policy rule to the storage.
func (a *adapter) AddPolicy(sec string, ptype string, rule []string) error {
	line := savePolicyLine(ptype, rule)

	if _, err := a.collection.InsertOne(context.Background(), line); err != nil {
		return err
	}

	return nil
}

// RemovePolicy removes a policy rule from the storage.
func (a *adapter) RemovePolicy(sec string, ptype string, rule []string) error {
	line := savePolicyLine(ptype, rule)

	if err := a.collection.Remove(context.Background(), bson.M{
		"ptype": line.PType,
		"v0":    line.V0,
		"v1":    line.V1,
		"v2":    line.V2,
		"v3":    line.V3,
		"v4":    line.V4,
		"v5":    line.V5,
	}); err != nil {
		if err == qmgo.ErrNoSuchDocuments {
			return nil
		}
		return err
	}

	return nil
}

// RemoveFilteredPolicy removes policy rules that match the filter from the storage.
func (a *adapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	selector := map[string]interface{}{
		"ptype": ptype,
	}
	if fieldIndex <= 0 && 0 < fieldIndex+len(fieldValues) {
		if fieldValues[0-fieldIndex] != "" {
			selector["v0"] = fieldValues[0-fieldIndex]
		}
	}
	if fieldIndex <= 1 && 1 < fieldIndex+len(fieldValues) {
		if fieldValues[1-fieldIndex] != "" {
			selector["v1"] = fieldValues[1-fieldIndex]
		}
	}
	if fieldIndex <= 2 && 2 < fieldIndex+len(fieldValues) {
		if fieldValues[2-fieldIndex] != "" {
			selector["v2"] = fieldValues[2-fieldIndex]
		}
	}
	if fieldIndex <= 3 && 3 < fieldIndex+len(fieldValues) {
		if fieldValues[3-fieldIndex] != "" {
			selector["v3"] = fieldValues[3-fieldIndex]
		}
	}
	if fieldIndex <= 4 && 4 < fieldIndex+len(fieldValues) {
		if fieldValues[4-fieldIndex] != "" {
			selector["v4"] = fieldValues[4-fieldIndex]
		}
	}
	if fieldIndex <= 5 && 5 < fieldIndex+len(fieldValues) {
		if fieldValues[5-fieldIndex] != "" {
			selector["v5"] = fieldValues[5-fieldIndex]
		}
	}

	if _, err := a.collection.RemoveAll(context.Background(), selector); err != nil {
		return err
	}

	return nil
}
