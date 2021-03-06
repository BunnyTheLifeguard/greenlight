package data

import (
	"context"
	"strings"
	"time"

	"github.com/BunnyTheLifeguard/greenlight/internal/validator"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Movie struct
type Movie struct {
	OID       primitive.ObjectID `json:"-" bson:"_id,omitempty"`
	ID        string             `json:"id,omitempty" bson:"id,omitempty"`
	CreatedAt time.Time          `json:"-"`
	Title     string             `json:"title" bson:"title,omitempty"`
	Year      int32              `json:"year,omitempty" bson:"year,omitempty"`
	Runtime   Runtime            `json:"runtime,omitempty" bson:"runtime,omitempty"`
	Genres    []string           `json:"genres,omitempty" bson:"genres,omitempty"`
	Version   int32              `json:"-"`
}

// MovieModel struct type wraps a MongoDB collection
type MovieModel struct {
	Collection *mongo.Collection
}

// ValidateMovie check for valid JSON
func ValidateMovie(v *validator.Validator, movie *Movie) {
	v.Check(movie.Title != "", "title", "must be provided")
	v.Check(len(movie.Title) <= 500, "title", "must not be more than 500 bytes long")

	v.Check(movie.Year != 0, "year", "must be provided")
	v.Check(movie.Year >= 1888, "year", "must be greater than 1888")
	v.Check(movie.Year <= int32(time.Now().Year()), "year", "must not be in the future")

	v.Check(movie.Runtime != 0, "runtime", "must be provided")
	v.Check(movie.Runtime > 0, "runtime", "must be a positive integer")

	v.Check(movie.Genres != nil, "genres", "must be provided")
	v.Check(len(movie.Genres) >= 1, "genres", "must contain at least 1 genre")
	v.Check(len(movie.Genres) <= 5, "genres", "must not contain more than 5 genres")
	v.Check(validator.Unique(movie.Genres), "genres", "must not contain duplicate values")
}

// Insert method for creating a new record
func (m MovieModel) Insert(movie *Movie) (string, error) {
	oid := primitive.NewObjectID()

	args := Movie{
		OID:       oid,
		ID:        oid.Hex(),
		CreatedAt: time.Now(),
		Title:     movie.Title,
		Year:      movie.Year,
		Runtime:   movie.Runtime,
		Genres:    movie.Genres,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := m.Collection.InsertOne(ctx, args)
	if err != nil {
		return "", err
	}

	filter := bson.M{"_id": oid}
	update := bson.M{"$inc": bson.M{"version": 1}}
	_ = m.Collection.FindOneAndUpdate(ctx, filter, update)
	return oid.Hex(), nil
}

// Get method for fetching a specific record
func (m MovieModel) Get(id string) (*Movie, error) {
	var result *Movie
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": oid}
	err = m.Collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Update method for editing a specific record
func (m MovieModel) Update(movie *Movie, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"title":   movie.Title,
			"year":    movie.Year,
			"runtime": movie.Runtime,
			"genres":  movie.Genres},
		"$inc": bson.M{"version": 1}}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = m.Collection.UpdateByID(ctx, oid, update)
	if err != nil {
		return err
	}

	return nil
}

// Delete method for removing a specific record
func (m MovieModel) Delete(id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	delete := bson.M{"_id": oid}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, _ := m.Collection.DeleteOne(ctx, delete)
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

// GetAll method to list of all records
func (m MovieModel) GetAll(title string, genres []string, filters Filters) ([]*Movie, Metadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var sort bson.D
	switch filters.Sort {
	case "title":
		sort = bson.D{{Key: "title", Value: 1}}
	case "-title":
		sort = bson.D{{Key: "title", Value: -1}}
	case "runtime":
		sort = bson.D{{Key: "runtime", Value: 1}}
	case "-runtime":
		sort = bson.D{{Key: "runtime", Value: -1}}
	case "year":
		sort = bson.D{{Key: "year", Value: 1}}
	case "-year":
		sort = bson.D{{Key: "year", Value: -1}}
	case "-id":
		sort = bson.D{{Key: "id", Value: -1}}
	default:
		sort = bson.D{{Key: "id", Value: 1}}
	}

	findOpts := options.Find().SetSort(sort).SetLimit(int64(filters.limit())).SetSkip(int64(filters.offset()))

	var filter bson.D
	if title != "" && len(genres) != 0 {
		searchSlice := append([]string{title}, genres...)
		searchString := strings.Join(searchSlice, ", ")
		filter = bson.D{{Key: "$text", Value: bson.D{{Key: "$search", Value: searchString}}}}
	} else if title != "" && len(genres) == 0 {
		filter = bson.D{{Key: "$text", Value: bson.D{{Key: "$search", Value: title}}}}
	} else if title == "" && len(genres) != 0 {
		searchString := strings.Join(genres, ", ")
		filter = bson.D{{Key: "$text", Value: bson.D{{Key: "$search", Value: searchString}}}}
	} else {
		filter = bson.D{}
	}

	countOpts := options.Count()
	if filters.limit() != 0 {
		countOpts = options.Count().SetLimit(int64(filters.limit())).SetSkip(int64(filters.offset()))
	}
	count, err := m.Collection.CountDocuments(ctx, filter, countOpts)
	if err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(int(count), filters.Page, filters.PageSize)

	cursor, err := m.Collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, Metadata{}, err
	}

	var results []*Movie
	if err = cursor.All(ctx, &results); err != nil {
		return nil, Metadata{}, err
	}

	return results, metadata, nil
}
