package engine

import "testing"

func TestHashTable_SetGet(t *testing.T) {
    ht := newHashTable()

    ht.Set("key1", "value1")
    ht.Set("key2", "value2")

    val, ok := ht.Get("key1")
    if !ok || val != "value1" {
        t.Errorf("Get(key1) = %v, %v; want value1, true", val, ok)
    }

    val, ok = ht.Get("key2")
    if !ok || val != "value2" {
        t.Errorf("Get(key2) = %v, %v; want value2, true", val, ok)
    }
}

func TestHashTable_Del(t *testing.T) {
    ht := newHashTable()

    ht.Set("key1", "value1")
    ht.Del("key1")

    _, ok := ht.Get("key1")
    if ok {
        t.Error("Get(key1) after delete should return false")
    }
}