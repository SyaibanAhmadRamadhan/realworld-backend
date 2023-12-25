package repository

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"realworld-go/domain"
	"realworld-go/domain/model"
)

type tagRepositoryImpl struct {
	db *mongo.Database
}

func NewTagRepositoryImpl(db *mongo.Database) domain.TagRepository {
	return &tagRepositoryImpl{
		db: db,
	}
}

func (t *tagRepositoryImpl) FindAllByNames(ctx context.Context, tagNames []string) (tags []model.Tag, err error) {
	filter := bson.D{
		bson.E{Key: "name", Value: bson.D{bson.E{Key: "$in", Value: tagNames}}},
	}

	cur, err := t.db.Collection(model.TagTableName).Find(ctx, filter)
	for cur.Next(ctx) {
		var tag model.Tag
		if err = cur.Decode(&tag); err != nil {
			return
		}

		tags = append(tags, tag)
	}

	return
}

func (t *tagRepositoryImpl) FindByName(ctx context.Context, name string) (tag model.Tag, err error) {
	filter := bson.D{
		bson.E{Key: "name", Value: name},
	}

	err = t.db.Collection(model.TagTableName).FindOne(ctx, filter).Decode(&tag)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			err = domain.ErrDataNotFound
		}
	}

	return
}

func (t *tagRepositoryImpl) FindTagPopuler(ctx context.Context, limit int64) (res []domain.FindTagPopulerResult, err error) {
	groupStage := bson.D{
		bson.E{Key: "$group", Value: bson.D{
			bson.E{Key: "_id", Value: "$tagID"},
			bson.E{Key: "count", Value: bson.D{
				bson.E{Key: "$sum", Value: 1},
			}},
		}},
	}
	projectStage := bson.D{
		{Key: "$project", Value: bson.D{
			{Key: "tagID", Value: "$_id"},
			{Key: "count", Value: "$count"},
			{Key: "_id", Value: 0},
		}},
	}
	sortStage := bson.D{
		bson.E{Key: "$sort", Value: bson.D{
			bson.E{Key: "count", Value: -1},
		}},
	}
	limitStage := bson.D{
		bson.E{Key: "$limit", Value: limit},
	}

	cur, err := t.db.Collection(model.ArticleTagTableName).Aggregate(ctx, mongo.Pipeline{
		groupStage, projectStage, sortStage, limitStage,
	})

	err = cur.All(ctx, &res)

	return
}

func (t *tagRepositoryImpl) UpSertMany(ctx context.Context, tagNames []string) (err error) {
	setMany := bson.D{}
	for _, tagName := range tagNames {
		setMany = append(setMany, bson.E{Key: "name", Value: tagName})
	}

	update := bson.D{{Key: "$set", Value: setMany}}
	_, err = t.db.Collection(model.TagTableName).UpdateMany(ctx, nil, update, options.Update().SetUpsert(true))
	if err != nil {
		return err
	}

	return
}

func (t *tagRepositoryImpl) DeleteByID(ctx context.Context, tag model.Tag) (err error) {
	filter := bson.D{bson.E{Key: "_id", Value: tag.ID}}

	res, err := t.db.Collection(model.TagTableName).DeleteOne(ctx, filter)
	if err != nil {
		return
	}

	if res.DeletedCount == 0 {
		return domain.ErrDelDataNotFound
	}

	return
}
