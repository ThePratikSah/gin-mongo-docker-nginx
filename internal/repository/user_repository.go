// Package repository implements persistence against MongoDB.
// BSON document shapes live here and never leak into other layers.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"ginmongo/internal/domain"
)

const usersCollection = "users"

type userDoc struct {
	ID        bson.ObjectID `bson:"_id"`
	Name      string        `bson:"name"`
	Email     string        `bson:"email"`
	CreatedAt time.Time     `bson:"created_at"`
	UpdatedAt time.Time     `bson:"updated_at"`
}

func (d userDoc) toDomain() domain.User {
	return domain.User{
		ID:        d.ID.Hex(),
		Name:      d.Name,
		Email:     d.Email,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

type UserRepository struct {
	coll *mongo.Collection
}

// NewUserRepository receives the database it should use; it never creates one.
func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{coll: db.Collection(usersCollection)}
}

// EnsureIndexes is idempotent; call it once at startup.
func (r *UserRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("uniq_email"),
	})
	if err != nil {
		return fmt.Errorf("create user indexes: %w", err)
	}
	return nil
}

func (r *UserRepository) Create(ctx context.Context, u domain.User) (domain.User, error) {
	doc := userDoc{
		ID:        bson.NewObjectID(),
		Name:      u.Name,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
	if _, err := r.coll.InsertOne(ctx, doc); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return domain.User{}, domain.ErrEmailTaken
		}
		return domain.User{}, fmt.Errorf("insert user: %w", err)
	}
	return doc.toDomain(), nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (domain.User, error) {
	oid, err := parseID(id)
	if err != nil {
		return domain.User{}, err
	}

	var doc userDoc
	err = r.coll.FindOne(ctx, bson.D{{Key: "_id", Value: oid}}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.User{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("find user: %w", err)
	}
	return doc.toDomain(), nil
}

func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]domain.User, error) {
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cur, err := r.coll.Find(ctx, bson.D{}, opts)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	var docs []userDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode users: %w", err)
	}

	out := make([]domain.User, 0, len(docs))
	for _, d := range docs {
		out = append(out, d.toDomain())
	}
	return out, nil
}

func (r *UserRepository) Update(ctx context.Context, id string, upd domain.UserUpdate) (domain.User, error) {
	oid, err := parseID(id)
	if err != nil {
		return domain.User{}, err
	}

	set := bson.D{{Key: "updated_at", Value: upd.UpdatedAt}}
	if upd.Name != nil {
		set = append(set, bson.E{Key: "name", Value: *upd.Name})
	}
	if upd.Email != nil {
		set = append(set, bson.E{Key: "email", Value: *upd.Email})
	}

	res := r.coll.FindOneAndUpdate(
		ctx,
		bson.D{{Key: "_id", Value: oid}},
		bson.D{{Key: "$set", Value: set}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var doc userDoc
	if err := res.Decode(&doc); err != nil {
		switch {
		case errors.Is(err, mongo.ErrNoDocuments):
			return domain.User{}, domain.ErrNotFound
		case mongo.IsDuplicateKeyError(err):
			return domain.User{}, domain.ErrEmailTaken
		default:
			return domain.User{}, fmt.Errorf("update user: %w", err)
		}
	}
	return doc.toDomain(), nil
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	oid, err := parseID(id)
	if err != nil {
		return err
	}

	res, err := r.coll.DeleteOne(ctx, bson.D{{Key: "_id", Value: oid}})
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if res.DeletedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func parseID(id string) (bson.ObjectID, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return bson.ObjectID{}, domain.ErrInvalidID
	}
	return oid, nil
}
