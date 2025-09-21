package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInsertSection(t *testing.T) {
	t.Run("create section", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)
		db := NewDB(gormDb)

		section := &Section{Name: "Trial"}
		err = db.InsertSection(section)
		a.Nil(err)

		dbSection := &Section{}
		tx := gormDb.First(dbSection, section.ID)
		a.Nil(tx.Error)
		a.Equal(section.Name, dbSection.Name)
		a.Len(dbSection.Modules, 0)
	})

	t.Run("duplicate name returns error", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)
		db := NewDB(gormDb)

		s1 := &Section{Name: "Trial"}
		err = db.InsertSection(s1)
		a.Nil(err)

		s2 := &Section{Name: "Trial"}
		err = db.InsertSection(s2)
		a.Error(err)
		a.Contains(err.Error(), "unique constraint")
	})
}

func TestInsertModule(t *testing.T) {
	t.Run("create module with section", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)
		db := NewDB(gormDb)

		section := &Section{Name: "Trial"}
		gormDb.Create(section)

		module := &Module{Name: "Trial", SectionID: section.ID}
		err = db.InsertModule(module)
		a.Nil(err)

		dbModule := &Module{}
		tx := gormDb.First(dbModule, module.ID)
		a.Nil(tx.Error)
		a.Equal(module.Name, dbModule.Name)
		a.Len(dbModule.Sensors, 0)
	})

	t.Run("same module name allowed in different sections", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)
		db := NewDB(gormDb)

		s1 := &Section{Name: "Trial1"}
		gormDb.Create(s1)
		s2 := &Section{Name: "Trial2"}
		gormDb.Create(s2)

		m1 := &Module{Name: "Trial", SectionID: s1.ID}
		err = db.InsertModule(m1)
		a.Nil(err)
		m2 := &Module{Name: "Trial", SectionID: s2.ID}
		err = db.InsertModule(m2)
		a.Nil(err)

		dbM1 := &Module{}
		tx := gormDb.First(dbM1, m1.ID)
		a.Nil(tx.Error)
		a.Equal(m1.Name, dbM1.Name)

		dbM2 := &Module{}
		tx = gormDb.First(dbM2, m2.ID)
		a.Nil(tx.Error)
		a.Equal(m2.Name, dbM2.Name)
	})

	t.Run("duplicate module name in same section returns error", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)
		db := NewDB(gormDb)

		s := &Section{Name: "Trial"}
		gormDb.Create(s)

		m1 := &Module{Name: "Trial", SectionID: s.ID}
		err = db.InsertModule(m1)
		a.Nil(err)

		m2 := &Module{Name: "Trial", SectionID: s.ID}
		err = db.InsertModule(m2)
		a.Error(err)
	})
}

func TestInsertSensor(t *testing.T) {
	t.Run("create sensor with module", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)
		db := NewDB(gormDb)

		s := &Section{Name: "Trial"}
		gormDb.Create(s)
		m := &Module{Name: "Trial", SectionID: s.ID}
		gormDb.Create(m)

		sensor := &Sensor{Name: "Trial", ModuleID: m.ID}
		err = db.InsertSensor(sensor)
		a.Nil(err)

		dbSensor := &Sensor{}
		tx := gormDb.First(dbSensor, sensor.ID)
		a.Nil(tx.Error)
		a.Equal(sensor.Name, dbSensor.Name)
		a.Len(dbSensor.Records, 0)
	})

	t.Run("duplicate sensor name in same module returns error", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)
		db := NewDB(gormDb)

		s := &Section{Name: "Trial"}
		gormDb.Create(s)
		m := &Module{Name: "Trial", SectionID: s.ID}
		gormDb.Create(m)

		sensor1 := &Sensor{Name: "Trial", ModuleID: m.ID}
		err = db.InsertSensor(sensor1)
		a.Nil(err)

		sensor2 := &Sensor{Name: "Trial", ModuleID: m.ID}
		err = db.InsertSensor(sensor2)
		a.Error(err)
	})
}

func TestInsertRecord(t *testing.T) {
	t.Run("create record for sensor", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)
		db := NewDB(gormDb)

		s := &Section{Name: "Trial"}
		gormDb.Create(s)
		m := &Module{Name: "Trial", SectionID: s.ID}
		gormDb.Create(m)
		sensor := &Sensor{Name: "Trial", ModuleID: m.ID}
		gormDb.Create(sensor)

		record := &Record{Value: 42, SensorID: sensor.ID}
		err = db.InsertRecord(record)
		a.Nil(err)

		dbRecord := &Record{}
		tx := gormDb.First(dbRecord, record.ID)
		a.Nil(tx.Error)
		a.Equal(record.Value, dbRecord.Value)
	})
}

func TestInsertUser(t *testing.T) {
	t.Run("create users with distinct tokens", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)
		db := NewDB(gormDb)

		u1 := &User{Username: "Apex1", Token: "Corse1"}
		err = db.InsertUser(u1)
		a.Nil(err)

		u2 := &User{Username: "Apex2", Token: "Corse2"}
		err = db.InsertUser(u2)
		a.Nil(err)

		users := make([]User, 0)
		gormDb.Find(&users)
		a.Len(users, 2)
	})

	t.Run("duplicate token returns error", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)
		db := NewDB(gormDb)

		u1 := &User{Username: "Apex1", Token: "Corse"}
		err = db.InsertUser(u1)
		a.Nil(err)

		u2 := &User{Username: "Apex2", Token: "Corse"}
		err = db.InsertUser(u2)
		a.Error(err)

		users := make([]User, 0)
		gormDb.Find(&users)
		a.Len(users, 1)
	})
}

func TestGetModuleById(t *testing.T) {
	t.Run("found returns module with sensors", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)
		db := NewDB(gormDb)

		s := &Section{Name: "trial"}
		gormDb.Create(s)
		m := &Module{Name: "trial", SectionID: s.ID}
		gormDb.Create(m)

		sensors := []*Sensor{
			{Name: "trial1", ModuleID: m.ID},
			{Name: "trial2", ModuleID: m.ID},
		}
		gormDb.Create(sensors)

		dbM, err := db.GetModuleById(m.ID)
		a.Nil(err)
		a.Equal(m.Name, dbM.Name)
		a.Len(dbM.Sensors, 2)
	})

	t.Run("not found returns error", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)
		db := NewDB(gormDb)

		s := &Section{Name: "trial"}
		gormDb.Create(s)
		m := &Module{Name: "trial", SectionID: s.ID}
		gormDb.Create(m)

		sensors := []*Sensor{
			{Name: "trial1", ModuleID: m.ID},
			{Name: "trial2", ModuleID: m.ID},
		}
		gormDb.Create(sensors)

		dbM, err := db.GetModuleById(999)
		a.Error(err)
		a.Nil(dbM)
	})
}

func TestGetSectionById(t *testing.T) {
	t.Run("found returns section with modules", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)
		db := NewDB(gormDb)

		s := &Section{Name: "Trial"}
		gormDb.Create(s)

		modules := []*Module{
			{Name: "Trial1", SectionID: s.ID},
			{Name: "Trial2", SectionID: s.ID},
			{Name: "Trial3", SectionID: s.ID},
		}
		gormDb.Create(modules)

		dbS, err := db.GetSectionById(s.ID)
		a.Nil(err)
		a.Equal(s.ID, dbS.ID)
		a.Len(dbS.Modules, 3)
	})
}

func TestGetSectionByName(t *testing.T) {
	t.Run("success returns section with modules", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)
		db := NewDB(gormDb)

		s := &Section{Name: "Trial"}
		gormDb.Create(s)

		modules := []*Module{
			{Name: "Trial1", SectionID: s.ID},
			{Name: "Trial2", SectionID: s.ID},
			{Name: "Trial3", SectionID: s.ID},
		}
		gormDb.Create(modules)

		dbS, err := db.GetSectionByName(s.Name)
		a.Nil(err)
		a.Equal(s.ID, dbS.ID)
		a.Len(dbS.Modules, 3)
	})

	t.Run("not found returns error", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)
		db := NewDB(gormDb)

		s := &Section{Name: "Trial"}
		gormDb.Create(s)
		modules := []*Module{
			{Name: "Trial1", SectionID: s.ID},
			{Name: "Trial2", SectionID: s.ID},
			{Name: "Trial3", SectionID: s.ID},
		}
		gormDb.Create(modules)

		dbS, err := db.GetSectionByName("42")
		a.Error(err)
		a.Nil(dbS)
	})
}

func TestGetModuleByNameAndSection(t *testing.T) {
	t.Run("success returns module", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)
		db := NewDB(gormDb)

		s := &Section{Name: "Trial"}
		gormDb.Create(s)
		m := &Module{Name: "Trial", SectionID: s.ID}
		gormDb.Create(m)

		dbM, err := db.GetModuleByNameAndSection(m.Name, s.Name)
		a.Nil(err)
		a.Equal(m.Name, dbM.Name)
	})

	t.Run("not found returns error", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)
		db := NewDB(gormDb)

		s := &Section{Name: "Trial"}
		gormDb.Create(s)
		m := &Module{Name: "Trial", SectionID: s.ID}
		gormDb.Create(m)

		dbM, err := db.GetModuleByNameAndSection(m.Name, "42")
		a.Error(err)
		a.Nil(dbM)
	})
}

func TestGetSensorById(t *testing.T) {
	t.Run("success returns sensor with records", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)
		db := NewDB(gormDb)

		s := &Section{Name: "Trial"}
		gormDb.Create(s)
		m := &Module{Name: "Trial", SectionID: s.ID}
		gormDb.Create(m)
		sensor := &Sensor{Name: "Trial", ModuleID: m.ID}
		gormDb.Create(sensor)

		records := []*Record{
			{Value: 42, SensorID: sensor.ID},
			{Value: 43, SensorID: sensor.ID},
			{Value: 44, SensorID: sensor.ID},
		}
		gormDb.Create(records)

		dbSensor, err := db.GetSensorById(sensor.ID, time.Time{}, time.Time{})
		a.Nil(err)
		a.Equal(sensor.Name, dbSensor.Name)
		a.Len(dbSensor.Records, 3)
	})
}

func TestGetSensorByNameAndModuleAndSection(t *testing.T) {
	t.Run("success returns sensor with ordered records", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)
		db := NewDB(gormDb)

		s := &Section{Name: "Trial"}
		gormDb.Create(s)
		m := &Module{Name: "Trial", SectionID: s.ID}
		gormDb.Create(m)
		sensor := &Sensor{Name: "Trial", ModuleID: m.ID}
		gormDb.Create(sensor)

		records := []*Record{
			{Value: 42, SensorID: sensor.ID},
			{Value: 43, SensorID: sensor.ID},
			{Value: 44, SensorID: sensor.ID},
		}
		gormDb.Create(records)

		dbSensor, err := db.GetSensorByNameAndModuleAndSection(sensor.Name, m.Name, s.Name, time.Time{}, time.Time{})
		a.Nil(err)
		a.Equal(sensor.Name, dbSensor.Name)
		a.Len(dbSensor.Records, 3)
		a.Equal(float32(42), dbSensor.Records[0].Value)
		a.Equal(float32(43), dbSensor.Records[1].Value)
		a.Equal(float32(44), dbSensor.Records[2].Value)
	})

	t.Run("not found returns error", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)
		db := NewDB(gormDb)

		s := &Section{Name: "Trial"}
		gormDb.Create(s)
		m := &Module{Name: "Trial", SectionID: s.ID}
		gormDb.Create(m)
		sensor := &Sensor{Name: "Trial", ModuleID: m.ID}
		gormDb.Create(sensor)

		records := []*Record{
			{Value: 42, SensorID: sensor.ID},
			{Value: 43, SensorID: sensor.ID},
			{Value: 44, SensorID: sensor.ID},
		}
		gormDb.Create(records)

		dbSensor, err := db.GetSensorByNameAndModuleAndSection(sensor.Name, m.Name, "42", time.Time{}, time.Time{})
		a.Error(err)
		a.Nil(dbSensor)
	})
}

func TestGetSensorByTopic(t *testing.T) {
	// Tests for GetSensorByTopic
	t.Run("found returns sensor", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)
		db := NewDB(gormDb)

		s := &Section{Name: "Trial"}
		gormDb.Create(s)
		m := &Module{Name: "Trial", SectionID: s.ID}
		gormDb.Create(m)
		sensor := &Sensor{Name: "Temp", ModuleID: m.ID, Topic: "home/room/temp"}
		gormDb.Create(sensor)

		dbSensor, err := db.GetSensorByTopic(sensor.Topic)
		a.Nil(err)
		a.Equal(sensor.Name, dbSensor.Name)
		a.Equal(sensor.Topic, dbSensor.Topic)
	})

	t.Run("not found returns error", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)
		db := NewDB(gormDb)

		dbSensor, err := db.GetSensorByTopic("nope")
		a.Error(err)
		a.Nil(dbSensor)
		a.Contains(err.Error(), "couldn't find sensor by topic 'nope'")
	})
}
func TestGetUserByToken(t *testing.T) {
	t.Run("success returns user", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)
		db := NewDB(gormDb)

		user := &User{Username: "Apex", Token: "Corse"}
		gormDb.Create(user)

		dbUser, err := db.GetUserByToken(user.Token)
		a.Nil(err)
		a.Equal(user.Username, dbUser.Username)
		a.Equal(user.Token, dbUser.Token)
	})

	t.Run("not found returns error", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)
		db := NewDB(gormDb)

		dbUser, err := db.GetUserByToken("nope")
		a.Error(err)
		a.Nil(dbUser)
	})
}
