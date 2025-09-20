package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInsertSection(t *testing.T) {
	t.Run("should create section", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)

		db := NewDB(gormDb)

		section := &Section{
			Name: "Trial",
		}
		err = db.InsertSection(section)
		a.Nil(err)

		dbSection := &Section{}
		tx := gormDb.First(dbSection, section.ID)

		a.Nil(tx.Error)
		a.Equal(section.Name, dbSection.Name)
		a.Len(dbSection.Modules, 0)
	})

	t.Run("cannot create two sections with same name", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)

		db := NewDB(gormDb)

		section1 := &Section{
			Name: "Trial",
		}
		err = db.InsertSection(section1)
		a.Nil(err)

		dbSection := &Section{}
		tx := gormDb.First(dbSection, section1.ID)

		a.Nil(tx.Error)
		a.Equal(section1.Name, dbSection.Name)
		a.Len(dbSection.Modules, 0)

		section2 := &Section{
			Name: "Trial",
		}
		err = db.InsertSection(section2)
		a.Error(err)
		a.Contains(err.Error(), "unique constraint")
	})
}

func TestInsertModule(t *testing.T) {
	t.Run("should create a module given a section", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)

		db := NewDB(gormDb)

		section := &Section{
			Name: "Trial",
		}
		gormDb.Create(section)

		module := &Module{
			Name:      "Trial",
			SectionID: section.ID,
		}
		err = db.InsertModule(module)
		a.Nil(err)

		dbModule := &Module{}
		tx := gormDb.First(dbModule, module.ID)

		a.Nil(tx.Error)
		a.Equal(module.Name, dbModule.Name)
		a.Len(dbModule.Sensors, 0)
	})

	t.Run("should create a module with same name but two different sections", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)

		db := NewDB(gormDb)

		section1 := &Section{
			Name: "Trial1",
		}
		gormDb.Create(section1)

		section2 := &Section{
			Name: "Trial2",
		}
		gormDb.Create(section2)

		module1 := &Module{
			Name:      "Trial",
			SectionID: section1.ID,
		}
		err = db.InsertModule(module1)
		a.Nil(err)

		module2 := &Module{
			Name:      "Trial",
			SectionID: section2.ID,
		}
		err = db.InsertModule(module2)
		a.Nil(err)

		dbModule1 := &Module{}
		tx := gormDb.First(dbModule1, module1.ID)

		a.Nil(tx.Error)
		a.Equal(module1.Name, dbModule1.Name)
		a.Len(dbModule1.Sensors, 0)

		dbModule2 := &Module{}
		tx = gormDb.First(dbModule2, module2.ID)

		a.Nil(tx.Error)
		a.Equal(module2.Name, dbModule2.Name)
		a.Len(dbModule2.Sensors, 0)
	})

	t.Run("can't create two modules with same section", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)

		db := NewDB(gormDb)

		section := &Section{
			Name: "Trial1",
		}
		gormDb.Create(section)

		module1 := &Module{
			Name:      "Trial",
			SectionID: section.ID,
		}
		err = db.InsertModule(module1)
		a.Nil(err)

		module2 := &Module{
			Name:      "Trial",
			SectionID: section.ID,
		}
		err = db.InsertModule(module2)
		a.Error(err)

		dbModule := &Module{}
		tx := gormDb.First(dbModule, module1.ID)

		a.Nil(tx.Error)
		a.Equal(module1.Name, dbModule.Name)
		a.Len(dbModule.Sensors, 0)
	})
}

func TestInsertSensor(t *testing.T) {
	t.Run("should create sensor with given section and module", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)

		db := NewDB(gormDb)

		section := &Section{
			Name: "Trial",
		}
		gormDb.Create(section)

		module := &Module{
			Name:      "Trial",
			SectionID: section.ID,
		}
		gormDb.Create(module)

		sensor := &Sensor{
			Name:     "Trial",
			ModuleID: module.ID,
		}
		err = db.InsertSensor(sensor)
		a.Nil(err)

		dbSensor := &Sensor{}
		tx := gormDb.First(dbSensor, sensor.ID)

		a.Nil(tx.Error)
		a.Equal(sensor.Name, dbSensor.Name)
		a.Len(dbSensor.Records, 0)
	})

	t.Run("can't create two sensors with same module", func(t *testing.T) {
		gormDb, cleanUp, err := TestDB()
		if err != nil {
			t.Fatal("cannot setup db")
		}
		defer cleanUp()

		a := assert.New(t)

		db := NewDB(gormDb)

		section := &Section{
			Name: "Trial",
		}
		gormDb.Create(section)

		module := &Module{
			Name:      "Trial",
			SectionID: section.ID,
		}
		gormDb.Create(module)

		sensor1 := &Sensor{
			Name:     "Trial",
			ModuleID: module.ID,
		}
		err = db.InsertSensor(sensor1)
		a.Nil(err)

		sensor2 := &Sensor{
			Name:     "Trial",
			ModuleID: module.ID,
		}
		err = db.InsertSensor(sensor2)
		a.Error(err)

		dbSensor := &Sensor{}
		tx := gormDb.First(dbSensor, sensor1.ID)

		a.Nil(tx.Error)
		a.Equal(sensor1.Name, dbSensor.Name)
		a.Len(dbSensor.Records, 0)
	})
}

func TestInsertRecord(t *testing.T) {
	gormDb, cleanUp, err := TestDB()
	if err != nil {
		t.Fatal("cannot setup db")
	}
	defer cleanUp()

	db := NewDB(gormDb)

	section := &Section{
		Name: "Trial",
	}
	gormDb.Create(section)

	module := &Module{
		Name:      "Trial",
		SectionID: section.ID,
	}
	gormDb.Create(module)

	sensor := &Sensor{
		Name:     "Trial",
		ModuleID: module.ID,
	}
	gormDb.Create(sensor)

	record := &Record{
		Value:    42,
		SensorID: sensor.ID,
	}
	err = db.InsertRecord(record)
	assert.Nil(t, err)

	dbRecord := &Record{}
	tx := gormDb.First(dbRecord, record.ID)

	assert.Nil(t, tx.Error)
	assert.Equal(t, record.Value, dbRecord.Value)
}

func TestInsertUser(t *testing.T) {
	gormDb, cleanUp, err := TestDB()
	if err != nil {
		t.Fatal("cannot setup db")
	}
	defer cleanUp()

	db := NewDB(gormDb)

	user := &User{
		Username: "Apex",
		Token:    "Corse",
	}
	err = db.InsertUser(user)
	assert.Nil(t, err)

	dbUser := &User{}
	gormDb.Where("token = ?", user.Token).First(dbUser)

	assert.Equal(t, user.Username, dbUser.Username)
	assert.Equal(t, user.Token, dbUser.Token)
}

func TestGetModuleById(t *testing.T) {
	gormDb, cleanUp, err := TestDB()
	if err != nil {
		t.Fatal("cannot setup db")
	}
	defer cleanUp()

	db := NewDB(gormDb)

	section := &Section{
		Name: "Trial",
	}
	gormDb.Create(section)

	module := &Module{
		Name:      "Trial",
		SectionID: section.ID,
	}
	gormDb.Create(module)

	sensors := []*Sensor{
		{
			Name:     "Trial1",
			ModuleID: module.ID,
		},
		{
			Name:     "Trial2",
			ModuleID: module.ID,
		},
	}
	gormDb.Create(sensors)

	dbModule, err := db.GetModuleById(module.ID)

	assert.Nil(t, err)
	assert.Equal(t, module.Name, dbModule.Name)
	assert.Len(t, dbModule.Sensors, 2)
}

func TestGetSectionById(t *testing.T) {
	gormDb, cleanUp, err := TestDB()
	if err != nil {
		t.Fatal("cannot setup db")
	}
	defer cleanUp()

	db := NewDB(gormDb)

	section := &Section{
		Name: "Trial",
	}
	gormDb.Create(section)

	modules := []*Module{
		{
			Name:      "Trial1",
			SectionID: section.ID,
		},
		{
			Name:      "Trial2",
			SectionID: section.ID,
		},
		{
			Name:      "Trial3",
			SectionID: section.ID,
		},
	}
	gormDb.Create(modules)

	dbSection, err := db.GetSectionById(section.ID)
	assert.Nil(t, err)
	assert.Equal(t, section.ID, dbSection.ID)
	assert.Len(t, dbSection.Modules, 3)
}

func TestGetSectionByName_Success(t *testing.T) {
	gormDb, cleanUp, err := TestDB()
	if err != nil {
		t.Fatal("cannot setup db")
	}
	defer cleanUp()

	db := NewDB(gormDb)

	section := &Section{
		Name: "Trial",
	}
	gormDb.Create(section)

	modules := []*Module{
		{
			Name:      "Trial1",
			SectionID: section.ID,
		},
		{
			Name:      "Trial2",
			SectionID: section.ID,
		},
		{
			Name:      "Trial3",
			SectionID: section.ID,
		},
	}
	gormDb.Create(modules)

	dbSection, err := db.GetSectionByName(section.Name)
	assert.Nil(t, err)
	assert.Equal(t, section.ID, dbSection.ID)
	assert.Len(t, dbSection.Modules, 3)
}

func TestGetSectionByName_NotFound(t *testing.T) {
	gormDb, cleanUp, err := TestDB()
	if err != nil {
		t.Fatal("cannot setup db")
	}
	defer cleanUp()

	db := NewDB(gormDb)

	section := &Section{
		Name: "Trial",
	}
	gormDb.Create(section)

	modules := []*Module{
		{
			Name:      "Trial1",
			SectionID: section.ID,
		},
		{
			Name:      "Trial2",
			SectionID: section.ID,
		},
		{
			Name:      "Trial3",
			SectionID: section.ID,
		},
	}
	gormDb.Create(modules)

	dbSection, err := db.GetSectionByName("42")
	assert.Error(t, err)
	assert.Nil(t, dbSection)
}

func TestGetModuleByNameAndSection_Success(t *testing.T) {
	gormDb, cleanUp, err := TestDB()
	if err != nil {
		t.Fatal("cannot setup db")
	}
	defer cleanUp()

	db := NewDB(gormDb)

	section := &Section{
		Name: "Trial",
	}
	gormDb.Create(section)

	module := &Module{
		Name:      "Trial",
		SectionID: section.ID,
	}
	gormDb.Create(module)

	dbModule, err := db.GetModuleByNameAndSection(module.Name, section.Name)

	assert.Nil(t, err)
	assert.Equal(t, module.Name, dbModule.Name)
}

func TestGetModuleByNameAndSection_Failure(t *testing.T) {
	gormDb, cleanUp, err := TestDB()
	if err != nil {
		t.Fatal("cannot setup db")
	}
	defer cleanUp()

	db := NewDB(gormDb)

	section := &Section{
		Name: "Trial",
	}
	gormDb.Create(section)

	module := &Module{
		Name:      "Trial",
		SectionID: section.ID,
	}
	gormDb.Create(module)

	dbModule, err := db.GetModuleByNameAndSection(module.Name, "42")

	assert.Error(t, err)
	assert.Nil(t, dbModule)
}

func TestGetSensorById(t *testing.T) {
	gormDb, cleanUp, err := TestDB()
	if err != nil {
		t.Fatal("cannot setup db")
	}
	defer cleanUp()

	db := NewDB(gormDb)

	section := &Section{
		Name: "Trial",
	}
	gormDb.Create(section)

	module := &Module{
		Name:      "Trial",
		SectionID: section.ID,
	}
	gormDb.Create(module)

	sensor := &Sensor{
		Name:     "Trial",
		ModuleID: module.ID,
	}
	gormDb.Create(sensor)

	records := []*Record{
		{
			Value:    42,
			SensorID: sensor.ID,
		},
		{
			Value:    43,
			SensorID: sensor.ID,
		},
		{
			Value:    44,
			SensorID: sensor.ID,
		},
	}
	gormDb.Create(records)

	dbSensor, err := db.GetSensorById(sensor.ID, time.Time{}, time.Time{})

	assert.Nil(t, err)
	assert.Equal(t, sensor.Name, dbSensor.Name)
	assert.Len(t, dbSensor.Records, 3)
}

func TestGetSensorByNameAndModuleAndSection_Success(t *testing.T) {
	gormDb, cleanUp, err := TestDB()
	if err != nil {
		t.Fatal("cannot setup db")
	}
	defer cleanUp()

	db := NewDB(gormDb)

	section := &Section{
		Name: "Trial",
	}
	gormDb.Create(section)

	module := &Module{
		Name:      "Trial",
		SectionID: section.ID,
	}
	gormDb.Create(module)

	sensor := &Sensor{
		Name:     "Trial",
		ModuleID: module.ID,
	}
	gormDb.Create(sensor)

	records := []*Record{
		{
			Value:    42,
			SensorID: sensor.ID,
		},
		{
			Value:    43,
			SensorID: sensor.ID,
		},
		{
			Value:    44,
			SensorID: sensor.ID,
		},
	}
	gormDb.Create(records)

	dbSensor, err := db.GetSensorByNameAndModuleAndSection(sensor.Name, module.Name, section.Name, time.Time{}, time.Time{})

	assert.Nil(t, err)
	assert.Equal(t, sensor.Name, dbSensor.Name)
	assert.Len(t, dbSensor.Records, 3)
	assert.Equal(t, float32(42), dbSensor.Records[0].Value)
	assert.Equal(t, float32(43), dbSensor.Records[1].Value)
	assert.Equal(t, float32(44), dbSensor.Records[2].Value)
}

func TestGetSensorByNameAndModuleAndSection_Failure(t *testing.T) {
	gormDb, cleanUp, err := TestDB()
	if err != nil {
		t.Fatal("cannot setup db")
	}
	defer cleanUp()

	db := NewDB(gormDb)

	section := &Section{
		Name: "Trial",
	}
	gormDb.Create(section)

	module := &Module{
		Name:      "Trial",
		SectionID: section.ID,
	}
	gormDb.Create(module)

	sensor := &Sensor{
		Name:     "Trial",
		ModuleID: module.ID,
	}
	gormDb.Create(sensor)

	records := []*Record{
		{
			Value:    42,
			SensorID: sensor.ID,
		},
		{
			Value:    43,
			SensorID: sensor.ID,
		},
		{
			Value:    44,
			SensorID: sensor.ID,
		},
	}
	gormDb.Create(records)

	dbSensor, err := db.GetSensorByNameAndModuleAndSection(sensor.Name, module.Name, "42", time.Time{}, time.Time{})

	assert.Error(t, err)
	assert.Nil(t, dbSensor)
}

func TestGetUserByToken_Success(t *testing.T) {
	gormDb, cleanUp, err := TestDB()
	if err != nil {
		t.Fatal("cannot setup db")
	}
	defer cleanUp()

	db := NewDB(gormDb)

	user := &User{
		Username: "Apex",
		Token:    "Corse",
	}
	gormDb.Create(user)

	dbUser, err := db.GetUserByToken(user.Token)

	assert.Nil(t, err)
	assert.Equal(t, user.Username, dbUser.Username)
	assert.Equal(t, user.Token, dbUser.Token)
}
