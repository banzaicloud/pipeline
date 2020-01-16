# Error Handling Guideline

This guideline collects the practices and coding standards everyone should follow when it comes to handling errors.


## Try to handle the error locally

Go's philosophy on errors is simple: handle them where they occur.

This is why you see code like this a lot:

```go
err := doSomething()
if err != nil {
	// some handling
}
```

The number one rule is: try to handle the error locally (by retrying, fixing the data, whatever makes sense in the context).

The "error handling" in most of the cases turns out to be bubbling up the error in the stack though.
The problem with that approach is that vital information (context) gets lost by just returning the error.
(This is a result of Go's oversimplification, which in this case is quite harmful)


## Avoid handling an error twice

To mitigate that problem you can see a lot of the following code in this project:

```go
err := doSomething()
if err != nil {
	log.WithSomeContext().Error(err)
	return err
}
```

This is however an anti-pattern because it leads to handling the error more than once. Even worse,
it usually generates more than one log event which makes debugging a painful process.

From now on using this anti-pattern should end up in automatic rejection of any PR.

TODO: write a linter which disallows this pattern


## Attach context to the error

As an alternative to the above anti-pattern, context and stack trace should be attached to errors using one of the following way:

```go

import (
	"emperror.dev/errors"
)

// ...

err := doSomething()

// Attach a message and stack trace to the error
// Note: this overwrites any previous stack trace
err = errors.Wrap(err, "some additional message")

// Attach a message and stack trace to the error
// Same as above with the exception that it does not override already existing stack trace
// Use this if a stack trace is already available
err = errors.WrapIf(err, "some additional message")

// Attach stack trace to the error without attaching a message
// Note: this overwrites any previous stack trace
err = errors.WithStack(err)

// Attach message to the error without attaching stack trace
err = errors.WithMessage(err, "some additional message")

// Attach arbitrary context (mostly key-value pairs) to the error
err = errors.WithDetails(err, "key1", "value1", "key2", "value2" /*,...*/)

// Combination of errors.WrapIf and errors.WithDetails
err = errors.WrapIfWithDetails(err, "key1", "value1", "key2", "value2" /*,...*/)
```

In low level code, using `errors` package is fine, as third-party packages rarely care about stack trace.
In other places `emperror` package is preferred, but it's up to the developer to decide.

**Note:** idiomatic Go error messages start with lower-cased letters.


## Final error handling

When the error cannot be dealt with and it reaches a final point where it has to be logged and/or returned to the user,
an `emperror.Handler` instance should be used. See examples in the `api` and `config` packages.


## Other error handling anti-patterns

### Starting error message with capital letter

According to the Go recommendation, idiomatic Go error messages start with lower-cased letters.


### Format error instead of attaching context

Instead of returning formatted error messages, like the following:

```go
    return errors.WrapIff(err, "something %s failed", important)
```

one should attach information as context:

```go
    return errors.WrapIfWithDetails(err, "something failed", "what", important)
```

There is one exception from this rule: when the error message is known to be returned to the user directly
(which itself should happen in very special cases only). Even then information should still be added as context as well:

```go
    return errors.WithDetails(errors.WrapIff(err, "something %s failed", important), "what", important)
```
