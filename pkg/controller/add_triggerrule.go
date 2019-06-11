package controller

import (
	"github.com/caitong93/kube-trigger/pkg/controller/triggerrule"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, triggerrule.Add)
}
