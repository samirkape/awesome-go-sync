// mparser-db is responsible for handling database related operation
// which may include connect, write, query
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

// WriteData uses mongodb's  InsertMany()  function to insert documents to a
// dbName database and CollectionName collection
func WriteData(client *mongo.Client, DbName string, CollectionName string, data []interface{}) *mongo.Collection {
	//Create a handle to the respective collection in the database.
	collection := client.Database(DbName).Collection(CollectionName)
	//Perform InsertMany operation & validate against the error.
	_, err := collection.InsertMany(context.TODO(), data)

	if err != nil {
		log.Fatal(err)
	}
	return collection
}

func UpdateData(client *mongo.Client, DbName string, CollectionName string, data []interface{}) *mongo.Collection {
	collection := client.Database(DbName).Collection(CollectionName)
	for _, p := range data {
		pkg := p.(Package)
		filter := bson.M{"name": pkg.Name}
		update := bson.M{"$max": bson.M{"stars": pkg.Stars}}

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

func RemoveDuplicates(client *mongo.Client, DB string) {
	collections := ListCollections(client, DB)
	for _, coll := range collections {
		FindDeleteDoc(client, DB, coll)
	}
}

func ListCollections(client *mongo.Client, DB string) []string {
	collections, err := client.Database(DB).ListCollectionNames(context.TODO(), bson.D{})
	if err != nil {
		log.Fatal(err)
	}
	return collections
}

func FindDoc(client *mongo.Client, DB string, Collection string) (Package, error) {
	//Define filter query for fetching specific document from collection
	filter := bson.D{} //bson.D{{}} specifies 'all documents'
	//Create a handle to the respective collection in the database.
	collection := client.Database(DB).Collection(Collection)
	//Perform Find operation & validate against the error.
	cur, findError := collection.Find(context.TODO(), filter)
	if findError != nil {
		return Package{}, findError
	}
	defer cur.Close(context.TODO())
	//Map result to slice
	for cur.Next(context.TODO()) {
		var t Package
		err := cur.Decode(&t)
		if err != nil {
			return Package{}, err
		} else {
			return t, nil
		}
	}
	return Package{}, nil
}

func FindDeleteDoc(client *mongo.Client, DB string, Collection string) error {
	//Define filter query for fetching specific document from collection
	filter := bson.D{} //bson.D{{}} specifies 'all documents'
	//Create a handle to the respective collection in the database.
	collection := client.Database(DB).Collection(Collection)
	//Perform Find operation & validate against the error.
	cur, findError := collection.Find(context.TODO(), filter)
	if findError != nil {
		return findError
	}
	defer cur.Close(context.TODO())
	namemap := make(map[string]struct{})
	var estruct struct{}
	//Map result to slice
	for cur.Next(context.TODO()) {
		t := Package{}
		err := cur.Decode(&t)
		if err != nil {
			return err
		}
		if _, ok := namemap[t.URL]; ok {
			DeleteOne(client, DB, Collection, t.Name) // TODO
		} else {
			namemap[t.URL] = estruct
		}
	}
	return nil
}

func DeleteOne(client *mongo.Client, DB string, Collection string, name string) error {
	//Define filter query for fetching specific document from collection

	// id, err := primitive.ObjectIDFromHex("_id")
	// if err != nil {
	// 	return err
	// }

	filter := bson.M{"name": name}

	//Create a handle to the respective collection in the database.
	collection := client.Database(DB).Collection(Collection)
	//Perform DeleteOne operation & validate against the error.
	_, err := collection.DeleteOne(context.TODO(), filter)
	if err != nil {
		return err
	}
	//Return success without any error.
	return nil
}

// DbConnect establish connection to mongodb cloud database for a given URI and
// returns *mongo.Client  which needs to be used for further operations on database.
func GetDbClient() *mongo.Client {
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

func PackagePreprocess(final []Package, title string, client *mongo.Client, DbName string) []interface{} {
	var data []interface{}
	for i := 0; i < len(final); i++ {
		e := final[i]
		data = append(data, e)
	}
	return data
}

func DBWrite(client *mongo.Client, categories Categories) {
	for i, category := range categories {
		title := category.Title
		fmt.Println(i)
		if title == "" || category.PackageDetails == nil {
			continue
		}
		data := PackagePreprocess(category.PackageDetails, title, client, DbName)
		WriteData(client, DbName, title, data)
	}
}

func DBUpdate(client *mongo.Client, categories Categories) {
	for i, category := range categories {
		title := category.Title
		fmt.Println(i)
		if title == "" || category.PackageDetails == nil {
			continue
		}
		data := PackagePreprocess(category.PackageDetails, title, client, DbName)
		UpdateData(client, DbName, title, data)
	}
}

// func findPackages(colName string) ([]Package, error) {
// 	// packageList will contains packages that are
// 	// requested by user by providing category number
// 	var packageList []Package
// 	var cur *mongo.Cursor
// 	var findError error

// 	// Get database name and client from config
// 	client := GetDbClient()
// 	DB := GetPackageDbName()

// 	// Get collection handle
// 	collection := client.Database(DB).Collection(colName)

// 	// bson.D{} specifies 'all documents'
// 	filter := bson.D{}

// 	// Find  all documents in the "Collection"
// 	cur, findError = collection.Find(context.TODO(), filter)

// 	if findError != nil {
// 		return nil, findError
// 	}

// 	defer cur.Close(context.TODO())

// 	//Map result to slice
// 	for cur.Next(context.TODO()) {
// 		p := Package{}
// 		err := cur.Decode(&p)
// 		if err != nil {
// 			return nil, err
// 		} else {
// 			packageList = append(packageList, p)
// 		}
// 	}
// 	return packageList, nil
// }
