package chaos

import (
	"context"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/filecoin-project/specs-actors/support/mock"
	atesting "github.com/filecoin-project/specs-actors/support/testing"
	"github.com/ipfs/go-cid"
)

func TestSingleton(t *testing.T) {
	receiver := atesting.NewIDAddr(t, 100)
	builder := mock.NewBuilder(context.Background(), receiver)

	rt := builder.Build(t)
	var a Actor

	msg := "constructor should not be called; the Chaos actor is a singleton actor"
	rt.ExpectAssertionFailure(msg, func() {
		rt.Call(a.Constructor, abi.Empty)
	})
	rt.Verify()
}

func TestCallerValidationNone(t *testing.T) {
	receiver := atesting.NewIDAddr(t, 100)
	builder := mock.NewBuilder(context.Background(), receiver)

	rt := builder.Build(t)
	var a Actor

	rt.Call(a.CallerValidation, &CallerValidationArgs{Branch: CallerValidationBranchNone})
	rt.Verify()
}

func TestCallerValidationIs(t *testing.T) {
	caller := atesting.NewIDAddr(t, 100)
	receiver := atesting.NewIDAddr(t, 101)
	builder := mock.NewBuilder(context.Background(), receiver)

	rt := builder.Build(t)
	rt.SetCaller(caller, builtin.AccountActorCodeID)
	var a Actor

	caddrs := []address.Address{atesting.NewIDAddr(t, 101)}

	rt.ExpectValidateCallerAddr(caddrs...)
	// FIXME: https://github.com/filecoin-project/specs-actors/pull/1155
	rt.ExpectAbort(exitcode.ErrForbidden, func() {
		rt.Call(a.CallerValidation, &CallerValidationArgs{
			Branch: CallerValidationBranchIsAddress,
			Addrs:  caddrs,
		})
	})
	rt.Verify()

	rt.ExpectValidateCallerAddr(caller)
	rt.Call(a.CallerValidation, &CallerValidationArgs{
		Branch: CallerValidationBranchIsAddress,
		Addrs:  []address.Address{caller},
	})
	rt.Verify()
}

func TestCallerValidationType(t *testing.T) {
	caller := atesting.NewIDAddr(t, 100)
	receiver := atesting.NewIDAddr(t, 101)
	builder := mock.NewBuilder(context.Background(), receiver)

	rt := builder.Build(t)
	rt.SetCaller(caller, builtin.AccountActorCodeID)
	var a Actor

	rt.ExpectValidateCallerType(builtin.CronActorCodeID)
	// FIXME: https://github.com/filecoin-project/specs-actors/pull/1155
	rt.ExpectAbort(exitcode.ErrForbidden, func() {
		rt.Call(a.CallerValidation, &CallerValidationArgs{
			Branch: CallerValidationBranchIsType,
			Types:  []cid.Cid{builtin.CronActorCodeID},
		})
	})
	rt.Verify()

	rt.ExpectValidateCallerType(builtin.AccountActorCodeID)
	rt.Call(a.CallerValidation, &CallerValidationArgs{
		Branch: CallerValidationBranchIsType,
		Types:  []cid.Cid{builtin.AccountActorCodeID},
	})
	rt.Verify()
}

func TestCallerValidationInvalidBranch(t *testing.T) {
	receiver := atesting.NewIDAddr(t, 100)
	builder := mock.NewBuilder(context.Background(), receiver)

	rt := builder.Build(t)
	var a Actor

	rt.ExpectAssertionFailure("invalid branch passed to CallerValidation", func() {
		rt.Call(a.CallerValidation, &CallerValidationArgs{Branch: -1})
	})
	rt.Verify()
}

func TestDeleteActor(t *testing.T) {
	receiver := atesting.NewIDAddr(t, 100)
	beneficiary := atesting.NewIDAddr(t, 101)
	builder := mock.NewBuilder(context.Background(), receiver)

	rt := builder.Build(t)
	var a Actor

	rt.ExpectValidateCallerAny()
	rt.ExpectDeleteActor(beneficiary)
	rt.Call(a.DeleteActor, &beneficiary)
	rt.Verify()
}

func TestMutateStateInTransaction(t *testing.T) {
	receiver := atesting.NewIDAddr(t, 100)
	builder := mock.NewBuilder(context.Background(), receiver)

	rt := builder.Build(t)
	var a Actor

	rt.ExpectValidateCallerAny()
	rt.StateCreate(&State{})

	val := "__mutstat test"
	rt.Call(a.MutateState, &MutateStateArgs{
		Value:  val,
		Branch: MutateInTransaction,
	})

	var st State
	rt.GetState(&st)

	if st.Value != val {
		t.Fatal("state was not updated")
	}

	rt.Verify()
}

func TestMutateStateAfterTransaction(t *testing.T) {
	receiver := atesting.NewIDAddr(t, 100)
	builder := mock.NewBuilder(context.Background(), receiver)

	rt := builder.Build(t)
	var a Actor

	rt.ExpectValidateCallerAny()
	rt.StateCreate(&State{})

	val := "__mutstat test"
	rt.Call(a.MutateState, &MutateStateArgs{
		Value:  val,
		Branch: MutateAfterTransaction,
	})

	var st State
	rt.GetState(&st)

	// state should be updated successfully _in_ the transaction but not outside
	if st.Value != val+"-in" {
		t.Fatal("state was not updated")
	}

	rt.Verify()
}

func TestMutateStateReadonly(t *testing.T) {
	receiver := atesting.NewIDAddr(t, 100)
	builder := mock.NewBuilder(context.Background(), receiver)

	rt := builder.Build(t)
	var a Actor

	rt.ExpectValidateCallerAny()
	rt.StateCreate(&State{})

	val := "__mutstat test"
	rt.Call(a.MutateState, &MutateStateArgs{
		Value:  val,
		Branch: MutateReadonly,
	})

	var st State
	rt.GetState(&st)

	if st.Value != "" {
		t.Fatal("state was not expected to be updated")
	}

	rt.Verify()
}

func TestMutateStateInvalidBranch(t *testing.T) {
	receiver := atesting.NewIDAddr(t, 100)
	builder := mock.NewBuilder(context.Background(), receiver)

	rt := builder.Build(t)
	var a Actor

	rt.ExpectValidateCallerAny()
	rt.ExpectAssertionFailure("unknown mutation type", func() {
		rt.Call(a.MutateState, &MutateStateArgs{Branch: -1})
	})
	rt.Verify()
}

func TestAbortWith(t *testing.T) {
	receiver := atesting.NewIDAddr(t, 100)
	builder := mock.NewBuilder(context.Background(), receiver)

	rt := builder.Build(t)
	var a Actor

	msg := "__test forbidden"
	rt.ExpectAbortContainsMessage(exitcode.ErrForbidden, msg, func() {
		rt.Call(a.AbortWith, &AbortWithArgs{
			Code:         exitcode.ErrForbidden,
			Message:      msg,
			Uncontrolled: false,
		})
	})
	rt.Verify()
}

func TestAbortWithUncontrolled(t *testing.T) {
	receiver := atesting.NewIDAddr(t, 100)
	builder := mock.NewBuilder(context.Background(), receiver)

	rt := builder.Build(t)
	var a Actor

	msg := "__test uncontrolled panic"
	rt.ExpectAssertionFailure(msg, func() {
		rt.Call(a.AbortWith, &AbortWithArgs{
			Message:      msg,
			Uncontrolled: true,
		})
	})
	rt.Verify()
}

func TestInspectRuntime(t *testing.T) {
	caller := atesting.NewIDAddr(t, 100)
	receiver := atesting.NewIDAddr(t, 101)
	builder := mock.NewBuilder(context.Background(), receiver)

	rt := builder.Build(t)
	rt.SetCaller(caller, builtin.AccountActorCodeID)
	rt.StateCreate(&State{})
	var a Actor

	rt.ExpectValidateCallerAny()
	ret := rt.Call(a.InspectRuntime, abi.Empty)
	rtr, ok := ret.(*InspectRuntimeReturn)
	if !ok {
		t.Fatal("invalid return value")
	}
	if rtr.Caller != caller {
		t.Fatal("unexpected runtime caller")
	}
	if rtr.Receiver != receiver {
		t.Fatal("unexpected runtime receiver")
	}
	rt.Verify()
}
