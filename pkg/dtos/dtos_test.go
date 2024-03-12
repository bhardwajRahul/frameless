package dtos_test

import (
	"context"
	"encoding/json"
	"go.llib.dev/frameless/pkg/dtos"
	"go.llib.dev/testcase/assert"
	"go.llib.dev/testcase/random"
	"strconv"
	"testing"
)

var rnd = random.New(random.CryptoSeed{})

var _ dtos.MP = dtos.P[Ent, EntDTO]{}

func TestM(t *testing.T) {
	ctx := context.Background()
	t.Run("mapping T to itself T, passthrough mode without registration", func(t *testing.T) {
		expEnt := Ent{V: rnd.Int()}
		gotEnt, err := dtos.Map[Ent](ctx, expEnt)
		assert.NoError(t, err)
		assert.Equal(t, expEnt, gotEnt)
	})
	t.Run("flat structures", func(t *testing.T) {
		m := EntMapping{}
		defer dtos.Register[Ent, EntDTO](m.ToDTO, m.ToEnt)()

		expEnt := Ent{V: rnd.Int()}
		expDTO := EntDTO{V: strconv.Itoa(expEnt.V)}

		dto, err := dtos.Map[EntDTO](ctx, expEnt)
		assert.NoError(t, err)
		assert.Equal(t, expDTO, dto)

		ent, err := dtos.Map[Ent](ctx, dto)
		assert.NoError(t, err)
		assert.Equal(t, expEnt, ent)
	})
	t.Run("nested structures", func(t *testing.T) {
		em := EntMapping{}
		nem := NestedEntMapping{}
		defer dtos.Register[Ent, EntDTO](em.ToDTO, em.ToEnt)()
		defer dtos.Register[NestedEnt, NestedEntDTO](nem.ToDTO, nem.ToEnt)()

		expEnt := NestedEnt{ID: rnd.String(), Ent: Ent{V: rnd.Int()}}
		expDTO := NestedEntDTO{ID: expEnt.ID, Ent: EntDTO{V: strconv.Itoa(expEnt.Ent.V)}}

		dto, err := dtos.Map[NestedEntDTO](ctx, expEnt)
		assert.NoError(t, err)
		assert.Equal(t, expDTO, dto)

		ent, err := dtos.Map[NestedEnt](ctx, dto)
		assert.NoError(t, err)
		assert.Equal(t, expEnt, ent)
	})
}

func TestMap(t *testing.T) {
	ctx := context.Background()
	t.Run("nil M given", func(t *testing.T) {
		_, err := dtos.Map[EntDTO, Ent](nil, Ent{V: rnd.Int()})
		assert.Error(t, err)
	})
	t.Run("happy", func(t *testing.T) {
		em := EntMapping{}
		defer dtos.Register[Ent, EntDTO](em.ToDTO, em.ToEnt)()
		expEnt := Ent{V: rnd.Int()}
		expDTO := EntDTO{V: strconv.Itoa(expEnt.V)}

		dto, err := dtos.Map[EntDTO](ctx, expEnt)
		assert.NoError(t, err)
		assert.Equal(t, expDTO, dto)

		ent, err := dtos.Map[Ent](ctx, dto)
		assert.NoError(t, err)
		assert.Equal(t, expEnt, ent)
	})
	t.Run("rainy", func(t *testing.T) {
		var (
			ent = Ent{V: rnd.Int()}
			dto = EntDTO{V: strconv.Itoa(ent.V)}
		)

		_, err := dtos.Map[EntDTO](ctx, ent)
		assert.ErrorIs(t, err, dtos.ErrNoMapping)

		_, err = dtos.Map[Ent](ctx, dto)
		assert.ErrorIs(t, err, dtos.ErrNoMapping)

		defer dtos.Register[Ent, EntDTO](EntMapping{}.ToDTO, EntMapping{}.ToEnt)()

		_, err = dtos.Map[EntDTO](ctx, ent)
		assert.NoError(t, err)
	})
	t.Run("ptr", func(t *testing.T) {
		defer dtos.Register[Ent, EntDTO](EntMapping{}.ToDTO, EntMapping{}.ToEnt)()

		expEnt := Ent{V: rnd.Int()}
		expDTO := EntDTO{V: strconv.Itoa(expEnt.V)}

		dto, err := dtos.Map[*EntDTO](ctx, expEnt)
		assert.NoError(t, err)
		assert.NotNil(t, dto)
		assert.Equal(t, expDTO, *dto)
	})
}

func ExampleRegister() {
	// JSONMapping will contain mapping from entities to JSON DTO structures.
	// registering Ent <---> EntDTO mapping
	_ = dtos.Register[Ent, EntDTO](
		EntMapping{}.ToDTO,
		EntMapping{}.ToEnt,
	)
	// registering NestedEnt <---> NestedEntDTO mapping, which includes the mapping of the nested entities
	_ = dtos.Register[NestedEnt, NestedEntDTO](
		NestedEntMapping{}.ToDTO,
		NestedEntMapping{}.ToEnt,
	)

	var v = NestedEnt{
		ID: "42",
		Ent: Ent{
			V: 42,
		},
	}

	ctx := context.Background()
	dto, err := dtos.Map[NestedEntDTO](ctx, v)
	if err != nil { // handle err
		return
	}

	_ = dto // data mapped into a DTO and now ready for marshalling
	/*
		NestedEntDTO{
			ID: "42",
			Ent: EntDTO{
				V: "42",
			},
		}
	*/

	data, err := json.Marshal(dto)
	if err != nil { // handle error
		return
	}

	_ = data
	/*
		{
			"id": "42",
			"ent": {
				"v": "42"
			}
		}
	*/

}

type Ent struct {
	V int
}

type EntDTO struct {
	V string `json:"v"`
}

type EntMapping struct{}

func (EntMapping) ToDTO(ctx context.Context, ent Ent) (EntDTO, error) {
	return EntDTO{V: strconv.Itoa(ent.V)}, nil
}

func (EntMapping) ToEnt(ctx context.Context, dto EntDTO) (Ent, error) {
	v, err := strconv.Atoi(dto.V)
	if err != nil {
		return Ent{}, err
	}
	return Ent{V: v}, nil
}

type NestedEnt struct {
	ID  string
	Ent Ent
}

type NestedEntDTO struct {
	ID  string `json:"id"`
	Ent EntDTO `json:"ent"`
}

type NestedEntMapping struct{}

func (NestedEntMapping) ToEnt(ctx context.Context, dto NestedEntDTO) (NestedEnt, error) {
	return NestedEnt{
		ID:  dto.ID,
		Ent: dtos.MustMap[Ent](ctx, dto.Ent),
	}, nil
}

func (NestedEntMapping) ToDTO(ctx context.Context, ent NestedEnt) (NestedEntDTO, error) {
	return NestedEntDTO{
		ID:  ent.ID,
		Ent: dtos.MustMap[EntDTO](ctx, ent.Ent),
	}, nil
}