package util

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMulticontext_AllCanceled(t *testing.T) {
	mc := NewMulticontext()

	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	err := mc.AddCtx(ctx1, ctx2)
	assert.NoError(t, err, "failed to register contexts")

	cancel1()
	cancel2()

	select {
	case <-mc.Ctx().Done():
		return
	case <-time.After(time.Millisecond):
		assert.Fail(t, "out context was not canceled")
	}
}

func TestMulticontext_ForceCanceled(t *testing.T) {
	mc := NewMulticontext()
	mc.Cancel()

	select {
	case <-mc.Ctx().Done():
		return
	case <-time.After(time.Millisecond):
		assert.Fail(t, "out context was not canceled")
	}
}

func TestMulticontext_ForceCanceledChilds(t *testing.T) {
	mc := NewMulticontext()

	ctx, _ := context.WithCancel(context.Background())
	err := mc.AddCtx(ctx)
	assert.NoError(t, err, "failed to register context")

	mc.Cancel()

	select {
	case <-mc.Ctx().Done():
		return
	case <-time.After(time.Millisecond):
		assert.Fail(t, "out context was not canceled")
	}
}

func TestMulticontext_PartiallyCanceled(t *testing.T) {
	mc := NewMulticontext()
	defer mc.Cancel()

	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	err := mc.AddCtx(ctx1, ctx2)
	assert.NoError(t, err, "failed to register contexts")

	cancel2()

	select {
	case <-mc.Ctx().Done():
		assert.Fail(t, "child context is active, but out context was canceled")
	case <-time.After(time.Millisecond):
		return
	}
}
