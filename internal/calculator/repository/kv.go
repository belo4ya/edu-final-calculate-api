package repository

import (
	"encoding/json"
	"fmt"

	"github.com/dgraph-io/badger/v4"
)

// Expression key constructors
func exprKey(id string) []byte {
	return []byte("expr:" + id)
}

func exprListKey(id string) []byte {
	return []byte("expr:list:" + id)
}

func exprListPrefix() []byte {
	return []byte("expr:list:")
}

func exprTasksPrefix(id string) []byte {
	return []byte("expr:" + id + ":tasks:")
}

func exprTaskKey(exprID, taskID string) []byte {
	return []byte("expr:" + exprID + ":tasks:" + taskID)
}

func exprFinalTaskKey(exprID, taskID string) []byte {
	return []byte("expr:" + exprID + ":final:" + taskID)
}

func exprFinalTaskPrefix(exprID string) []byte {
	return []byte("expr:" + exprID + ":final:")
}

// Task key constructors
func taskKey(id string) []byte {
	return []byte("task:" + id)
}

func taskQueuePendingPrefix() []byte {
	return []byte("task:queue:pending:")
}

func taskQueuePendingKey(id string) []byte {
	return []byte("task:queue:pending:" + id)
}

func taskChildPrefix(id string) []byte {
	return []byte("task:" + id + ":child:")
}

func taskChildKey(id string, childID string) []byte {
	return []byte("task:" + id + ":child:" + childID)
}

// ID extraction from keys
func exprIDFromListKey(key []byte) string {
	return string(key)[len("expr:list:"):]
}

func taskIDFromExprTaskKey(key []byte, exprID string) string {
	return string(key)[len("expr:"+exprID+":tasks:"):]
}

func taskIDFromPendingQueueKey(key []byte) string {
	return string(key)[len("task:queue:pending:"):]
}

func taskIDFromExprFinalTaskKey(key []byte, exprID string) string {
	return string(key)[len("expr:"+exprID+":final:"):]
}

func taskIDFromTaskChildKey(key []byte, taskID string) string {
	return string(key)[len("task:"+taskID+":child:"):]
}

// BadgerDB operation helpers
func scanVal[T any](txn *badger.Txn, key []byte, dst *T) error {
	item, err := txn.Get(key)
	if err != nil {
		return fmt.Errorf("get %q: %w", string(key), err)
	}
	if err := item.Value(func(val []byte) error {
		return json.Unmarshal(val, dst)
	}); err != nil {
		return fmt.Errorf("json unmarhal %q: %w", string(key), err)
	}
	return nil
}

func setVal[T any](txn *badger.Txn, key []byte, val T) error {
	exprData, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("json marhal %q: %w", string(key), err)
	}
	if err := txn.Set(key, exprData); err != nil {
		return fmt.Errorf("set %q: %w", string(key), err)
	}
	return nil
}

func setOnlyKey(txn *badger.Txn, key []byte) error {
	if err := txn.Set(key, []byte{1}); err != nil {
		return fmt.Errorf("set only key %q: %w", string(key), err)
	}
	return nil
}
