package parser

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const DbName = "packagedb"

func updateCollection(client *mongo.Client, DbName string, CollectionName string, data interface{}) *mongo.Collection {
	collection := client.Database(DbName).Collection(CollectionName)
	for _, pkg := range data.([]Package) {
		filter := bson.M{"name": pkg.URL}
		update := bson.M{"$set": pkg}

		// Create an instance of an options and set the desired options.
		opt := options.Update().SetUpsert(true)

		result, err := collection.UpdateOne(context.Background(), filter, update, opt)
		if err != nil {
			log.Printf("repo star update failed: %v\n", err)
		} else if result.UpsertedCount > 0 {
			log.Printf("creating new entry for: %s", pkg.Name)
		} else {
			log.Printf("updating star count for: %s", pkg.Name)
		}
	}
	return collection
}

func removeDuplicates() {
	client := getClient()
	DB := DbName
	collections := listCollections(client, DB)
	for _, collection := range collections {
		err := findDeleteDoc(client, DB, collection)
		if err != nil {
			return
		}
	}
}

func listCollections(client *mongo.Client, DB string) []string {
	collections, err := client.Database(DB).ListCollectionNames(context.TODO(), bson.D{})
	if err != nil {
		log.Fatal(err)
	}
	return collections
}

func findDeleteDoc(client *mongo.Client, DB string, Collection string) error {
	//Define filter query for fetching specific document from collection
	filter := bson.D{} //bson.D{{}} specifies 'all documents'
	//Create a handle to the respective collection in the database.
	collection := client.Database(DB).Collection(Collection)
	//Perform Find operation & validate against the error.
	cur, findError := collection.Find(context.TODO(), filter)
	if findError != nil {
		return findError
	}
	defer func(cur *mongo.Cursor, ctx context.Context) {
		err := cur.Close(ctx)
		if err != nil {
			log.Println("error closing cursor", err)
			return
		}
	}(cur, context.TODO())
	urls := make(map[string]struct{})
	//Map result to slice
	for cur.Next(context.TODO()) {
		t := Package{}
		err := cur.Decode(&t)
		if err != nil {
			return err
		}
		if _, ok := urls[t.URL]; ok {
			err := deleteOneDoc(client, DB, Collection, t.Name)
			if err != nil {
				return err
			} // TODO
		} else {
			urls[t.URL] = struct{}{}
		}
	}
	return nil
}

func deleteOneDoc(client *mongo.Client, DB string, Collection string, name string) error {
	filter := bson.M{"name": name}
	//Create a handle to the respective collection in the database.
	collection := client.Database(DB).Collection(Collection)
	//Perform deleteOneDoc operation & validate against the error.
	_, err := collection.DeleteOne(context.TODO(), filter)
	if err != nil {
		return err
	}
	log.Printf("Deleted duplicate document with name: %v\n", name)
	return nil
}

func getClient() *mongo.Client {
	client, err := mongo.NewClient(options.Client().ApplyURI(Config.MongoURL))
	if err != nil {
		log.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return client
}

func writePackages(client *mongo.Client, categories Categories) {
	for i, category := range categories {
		title := category.Title
		fmt.Println(i)
		if title == "" || category.PackageDetails == nil {
			continue
		}
		updateCollection(client, DbName, title, category.PackageDetails)
	}
	removeDuplicates()
}
