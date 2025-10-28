package examples

import (
	"github.com/Emptyless/go-testbuilder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"testing"
)

func TestUserController_Handle(t *testing.T) {
	t.Parallel()

	type State struct {
		ctrl     *gomock.Controller
		userName string
		payload  string
		user     User
	}

	// builder
	builder := testbuilder.TestsBuilder[
		UserController,
		State,
		func(t *testing.T, controller UserController, state State, user *User, error error),
	]{}

	builder.Register("invalid payload").
		WithSpecificBuilder(func(t *testing.T, sut *UserController, state *State) {
			state.payload = ""
		}).
		WithAssertion(func(t *testing.T, controller UserController, state State, user *User, err error) {
			assert.Nil(t, user)
			require.EqualError(t, err, "invalid payload")
		})

	builder.Register("get user failure").
		WithStateBuilder(func(t *testing.T, sut *UserController, state *State) {
			state.ctrl = gomock.NewController(t)
			sut.Repository = NewMockUserRepository(state.ctrl)

			state.userName = "my-user"
			state.payload = "my-payload"
		}).
		WithSpecificBuilder(func(t *testing.T, sut *UserController, state *State) {
			sut.Repository.(*MockUserRepository).
				EXPECT().
				GetUser(state.userName).
				Return(User{}, assert.AnError)
		}).
		WithAssertion(func(t *testing.T, controller UserController, state State, user *User, err error) {
			assert.Nil(t, user)
			require.EqualError(t, err, assert.AnError.Error())
		})

	builder.Register("send mail failure").
		WithStateBuilder(func(t *testing.T, sut *UserController, state *State) {
			sut.Mailer = NewMockMailService(state.ctrl)
			state.userName = "my-user"
			state.payload = "my-payload"
			state.user = User{Name: state.userName}

			sut.Repository.(*MockUserRepository).
				EXPECT().
				GetUser(state.userName).
				Return(state.user, nil)
		}).
		WithSpecificBuilder(func(t *testing.T, sut *UserController, state *State) {
			sut.Mailer.(*MockMailService).
				EXPECT().
				SendMail().
				Return(assert.AnError)
		}).
		WithAssertion(func(t *testing.T, controller UserController, state State, user *User, err error) {
			assert.Nil(t, user)
			require.EqualError(t, err, assert.AnError.Error())
		})

	builder.Register("store user failure").
		WithStateBuilder(func(t *testing.T, sut *UserController, state *State) {
			sut.Mailer.(*MockMailService).
				EXPECT().
				SendMail().
				Return(nil)
		}).
		WithSpecificBuilder(func(t *testing.T, sut *UserController, state *State) {
			sut.Repository.(*MockUserRepository).
				EXPECT().
				StoreUser(state.user).
				Return(assert.AnError)
		}).
		WithAssertion(func(t *testing.T, controller UserController, state State, user *User, err error) {
			assert.Nil(t, user)
			require.EqualError(t, err, assert.AnError.Error())
		})

	builder.Register("success").
		WithStateBuilder(func(t *testing.T, sut *UserController, state *State) {
			sut.Repository.(*MockUserRepository).
				EXPECT().
				StoreUser(state.user).
				Return(nil)
		}).
		WithAssertion(func(t *testing.T, controller UserController, state State, user *User, err error) {
			require.Nil(t, err)
			require.NotNil(t, user)
			assert.Equal(t, state.user, *user)
		})

	// Run all test cases
	for name, buildTest := range builder.Tests() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// Arrange
			testData := buildTest(t)
			ctrl := testData.SUT

			// Act
			user, err := ctrl.Handle(testData.State.userName, testData.State.payload)

			// Assert
			testData.Assert(t, ctrl, testData.State, user, err)
		})
	}
}

