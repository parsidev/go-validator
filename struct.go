package validation

import "github.com/go-playground/validator/v10"

type StructLevel = validator.StructLevel

type StructLevelFunc = func(sl StructLevel)
