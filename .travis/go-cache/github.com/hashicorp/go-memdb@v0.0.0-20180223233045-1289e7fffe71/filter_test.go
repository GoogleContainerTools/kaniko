package memdb

import "testing"

// Test that the iterator meets the required interface
func TestFilterIterator_Interface(t *testing.T) {
	var _ ResultIterator = &FilterIterator{}
}

func TestFilterIterator(t *testing.T) {
	db := testDB(t)
	txn := db.Txn(true)

	obj1 := &TestObject{
		ID:  "a",
		Foo: "xyz",
		Qux: []string{"xyz1"},
	}
	obj2 := &TestObject{
		ID:  "medium-length",
		Foo: "xyz",
		Qux: []string{"xyz1", "xyz2"},
	}
	obj3 := &TestObject{
		ID:  "super-long-unique-identifier",
		Foo: "xyz",
		Qux: []string{"xyz1", "xyz2"},
	}

	err := txn.Insert("main", obj1)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	err = txn.Insert("main", obj2)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	err = txn.Insert("main", obj3)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	filterFactory := func(idLengthLimit int) func(interface{}) bool {
		limit := idLengthLimit
		return func(raw interface{}) bool {
			obj, ok := raw.(*TestObject)
			if !ok {
				return true
			}

			return len(obj.ID) > limit
		}
	}

	checkResult := func(txn *Txn) {
		// Attempt a row scan on the ID
		result, err := txn.Get("main", "id")
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		// Wrap the iterator and try various filters
		filter := NewFilterIterator(result, filterFactory(6))
		if raw := filter.Next(); raw != obj1 {
			t.Fatalf("bad: %#v %#v", raw, obj1)
		}

		if raw := filter.Next(); raw != nil {
			t.Fatalf("bad: %#v %#v", raw, nil)
		}

		result, err = txn.Get("main", "id")
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		filter = NewFilterIterator(result, filterFactory(15))
		if raw := filter.Next(); raw != obj1 {
			t.Fatalf("bad: %#v %#v", raw, obj1)
		}

		if raw := filter.Next(); raw != obj2 {
			t.Fatalf("bad: %#v %#v", raw, obj2)
		}

		if raw := filter.Next(); raw != nil {
			t.Fatalf("bad: %#v %#v", raw, nil)
		}

		result, err = txn.Get("main", "id")
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		filter = NewFilterIterator(result, filterFactory(150))
		if raw := filter.Next(); raw != obj1 {
			t.Fatalf("bad: %#v %#v", raw, obj1)
		}

		if raw := filter.Next(); raw != obj2 {
			t.Fatalf("bad: %#v %#v", raw, obj2)
		}

		if raw := filter.Next(); raw != obj3 {
			t.Fatalf("bad: %#v %#v", raw, obj3)
		}

		if raw := filter.Next(); raw != nil {
			t.Fatalf("bad: %#v %#v", raw, nil)
		}
	}

	// Check the results within the txn
	checkResult(txn)

	// Commit and start a new read transaction
	txn.Commit()
	txn = db.Txn(false)

	// Check the results in a new txn
	checkResult(txn)
}
