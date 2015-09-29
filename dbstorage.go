package kwiscale

import "time"

/*
Must work like this:

user := &User{}

kwiscale.Datastore().Get(map[string]interface{}{
	"name" : "Foo"
}).Limit(10).Find(u)

*/

var dbdrivers = make(map[string]DB)

func RegisterDatastore(name string, ds DB) {
	dbdrivers[name] = ds
}

type DBModel struct {
	ID        int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (model *DBModel) OnCreate() {
	model.CreatedAt = time.Now()
}

func (model *DBModel) OnUpdate() {
	model.UpdatedAt = time.Now()
}

type DBOptions map[string]interface{}

type DB interface {
	SetOptions(DBOptions)
	Init()
	Insert(what interface{}) error
	Get(where map[string]interface{}) DBQuery
	Update(where map[string]interface{}, what interface{}) error
	Delete(what interface{}) error
	Close()
}

type DBQuery interface {
	Limit(int64) DBQuery
	Offset(int64) DBQuery
	One(interface{}) error
	All(interface{}) error
}
