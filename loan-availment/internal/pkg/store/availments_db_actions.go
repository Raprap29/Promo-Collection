package store

import (
	"context"
	"globe/dodrio_loan_availment/internal/pkg/consts"
	"globe/dodrio_loan_availment/internal/pkg/logger"
	"time"

	storeModels "globe/dodrio_loan_availment/internal/pkg/store/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type AvailmentStore struct {
	AvailmentStore *mongo.Collection
}

func NewAvailmentStore(mongoClient mongo.Client, DBName string) *AvailmentStore {
	Availment := mongoClient.Database(DBName).Collection(consts.AvailmentCollection)
	return &AvailmentStore{
		AvailmentStore: Availment,
	}
}

func (availmentStore AvailmentStore) GetFailedKafkaEntries(ctx context.Context, duration int32) ([]storeModels.AvailmentTransaction, error) {
	// Calculate the threshold time

	thresholdTime := time.Now().Add(-time.Duration(duration) * time.Hour)

	pipeline := mongo.Pipeline{
		{
			{"$match", bson.D{
				{"publishedToKafka", false},
				{"createdAt", bson.D{{"$gte", thresholdTime}}},
			}},
		},
		{
			{"$lookup", bson.D{
				{"from", "LoanProducts"},
				{"localField", "productId"},
				{"foreignField", "_id"},
				{"as", "productDetails"},
			}},
		},
		{
			{"$unwind", bson.D{
				{"path", "$productDetails"},
				{"preserveNullAndEmptyArrays", true},
			}},
		},
		{
			{"$lookup", bson.D{
				{"from", "Collections"},
				{"localField", "_id"},
				{"foreignField", "availmentTransactionId"},
				{"as", "collectionDetails"},
			}},
		},
		{
			{"$unwind", bson.D{
				{"path", "$collectionDetails"},
				{"preserveNullAndEmptyArrays", true},
			}},
		},
		{
			{"$project", bson.D{
				{"_id", 1},
				{"GUID", 1},
				{"productId", 1},
				{"brand", 1},
				{"MSISDN", 1},
				{"createdAt", 1},
				{"channel", 1},
				{"loanType", 1},
				{"keyword", 1},
				{"servicingPartner", 1},
				{"totalLoanAmount", 1},
				{"serviceFee", 1},
				{"errorText", 1},
				{"tokenPaymentId", 1},
				{"result", 1},
				{"publishedToKafka", 1},
				{"productName", "$productDetails.name"},
				{"loanStatus", "$collectionDetails.method"},
				{"collectionType", "$collectionDetails.collectionType"},
			}},
		},
	}

	// Execute the aggregation pipeline
	cursor, err := availmentStore.AvailmentStore.Aggregate(ctx, pipeline)
	if err != nil {
		logger.Error(ctx, "error during aggregation: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	// Decode the results
	var availments []storeModels.AvailmentTransaction
	if err := cursor.All(ctx, &availments); err != nil {
		logger.Error(ctx, "error decoding results: %v", err)
		return nil, err
	}
	return availments, nil
}

func (cs *AvailmentStore) SetKafkaFlag(ctx context.Context, AvailmentIds []string) ([]string, error) {
	// Convert AvailmentIds from string to primitive.ObjectID
	objectIDs := make([]primitive.ObjectID, len(AvailmentIds))
	for i, id := range AvailmentIds {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return nil, err
		}
		objectIDs[i] = objectID
	}
	// Create filter and update documents
	filter := bson.M{"_id": bson.M{"$in": objectIDs}}
	update := bson.M{"$set": bson.M{"publishedToKafka": true}}

	// Perform the update operation
	updateResult, err := cs.AvailmentStore.UpdateMany(ctx, filter, update)
	if err != nil {
		return nil, err
	}
	logger.Debug(ctx, "Matched Count: %d, Modified Count: %d", updateResult.MatchedCount, updateResult.ModifiedCount)

	// Check for documents that were not updated
	failedUpdates := []string{}
	if updateResult.MatchedCount != updateResult.ModifiedCount {
		// Find the documents that were not updated
		filterFailed := bson.M{
			"_id":              bson.M{"$in": objectIDs},
			"publishedToKafka": bson.M{"$ne": true}, // Not equal to true
		}
		cursor, err := cs.AvailmentStore.Find(ctx, filterFailed)
		if err != nil {
			return nil, err
		}
		defer cursor.Close(ctx)

		for cursor.Next(ctx) {
			var doc bson.M
			if err := cursor.Decode(&doc); err != nil {
				return nil, err
			}
			failedUpdates = append(failedUpdates, doc["_id"].(primitive.ObjectID).Hex())
		}
	}
	return failedUpdates, nil
}

// func AvailmentsInsertion(session sessions.Session) (string, error) {

// 	// log.Println(" I am inside AvailmentsInsertion ")

// 	msisdn := session.Get("msisdn")
// 	//brand := session.Get("brand").(string) //will come from UUP
// 	brand := "BR002"
// 	loanType := session.Get("loanType").(string)
// 	productId := session.Get("productId").(primitive.ObjectID)
// 	keyword := session.Get("keyword").(string)
// 	// keywordId := session.Get("keywordId").(string)
// 	servicingPartner := "Globe Wallet"
// 	//result := session.Get("result").(string)
// 	// result := "Success"
// 	//errorString := session.Get("errorString").(*string)
// 	creditScore := session.Get("creditScore").(int32)
// 	//status := session.Get("status").(bool)
// 	// status := true
// 	loanAmount := session.Get("loanAmount").(int32)
// 	serviceFee := session.Get("serviceFee").(int32)
// 	//channelId := session.Get("channel").(primitive.ObjectID)
// 	guid := session.Get("systemClient").(string)

// 	fmt.Println(loanType, loanAmount, serviceFee)

// 	db_name := configs.DB_NAME

// 	database := db.DbConnection()
// 	sampleMflixDatabase := database.Database(db_name)

// 	//Brands collection
// 	brandsCollection := sampleMflixDatabase.Collection("Brands")

// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()

// 	filter := bson.M{"visible": true}
// 	//filter := bson.M{"code": brand}
// 	//projection := bson.M{"_id": 1}

// 	var b models.Brand
// 	//err := brandsCollection.FindOne(context.Background(), filter, options.FindOne().SetProjection(projection)).Decode(&b)
// 	err := brandsCollection.FindOne(ctx, filter).Decode(&b)
// 	if err != nil {
// 		return "", fmt.Errorf("error while fetching id from brands collection %v", err.Error())
// 	}
// 	id := b.Id

// 	// log.Println("success queried Brands")

// 	// channel := "507f1f77bcf86cd799439202"

// 	// channelId, err_p := primitive.ObjectIDFromHex(channel)
// 	// if err_p != nil {
// 	// 	return "", fmt.Errorf("error while converting channel id to object id %v", err_p.Error())
// 	// }

// 	//Loan Products Channel Collection
// 	loanProductsChannelCollection := sampleMflixDatabase.Collection("LoanProducts_Channels")
// 	filter = bson.M{"productId": productId}
// 	channel := ""
// 	//projection := bson.M{"channelId": channelId}

// 	// log.Printf("productId passed in LoanProducts_Channels is: %v", productId)

// 	var l bson.M
// 	err = loanProductsChannelCollection.FindOne(ctx, filter).Decode(&l)
// 	if err != nil {
// 		return "", fmt.Errorf("error while fetching id from loanproducts channel collection %v", err.Error())
// 	}

// 	// log.Printf("LoanProducts_Channels: %v", l)

// 	// Extract the channelId
// 	channelID, ok := l["channelId"].(primitive.ObjectID)
// 	if !ok {
// 		return "", fmt.Errorf("channelId is not an ObjectID")
// 	}

// 	// Convert the ObjectID to a string
// 	channel = channelID.Hex()
// 	// log.Printf("channel fetched from LoanProducts_Channels: %v", channel)
// 	// channelId, _ := primitive.ObjectIDFromHex(channel)
// 	//channelId := l["channelId"].(primitive.ObjectID)

// 	// log.Println("success queried LoanProducts_Channels")

// 	//Availments collection
// 	availmentsCollection := sampleMflixDatabase.Collection("Availments")
// 	// record := models.Availments{MSISDN: msisdn, Channel: channelId, Brand: id, LoanType: loanType, ProductId: productId, Keyword: keywordId, ServicingPartner: servicingPartner, Result: result, ErrorString: nil, CreditScore: creditScore, Status: status, LoanAmount: loanAmount, ServiceFee: serviceFee, GUID: guid, CreatedAt: time.Now()}
// 	record := models.Availments{
// 		MSISDN:           msisdn.(string),
// 		Channel:          channel,
// 		Brand:            brand,
// 		LoanType:         loanType,
// 		ProductID:        productId,
// 		Keyword:          keyword,
// 		ServicingPartner: servicingPartner, // need to change
// 		Result:           true,
// 		ErrorText:        "",
// 		CreditScore:      creditScore, // take value from getDetailsByAttribute
// 		TotalLoanAmount:  loanAmount,  // need to change
// 		ServiceFee:       serviceFee,  // need to change
// 		GUID:             guid,
// 		CreatedAt:        time.Now(),
// 		PublishedToKafka: true,
// 	}

// 	// log.Println("Going to insert successful data in Availment")

// 	availmentsInsertRecord, err := availmentsCollection.InsertOne(context.Background(), record)
// 	log.Println("Availment record successfully inserted")

// 	if err != nil {
// 		return "", fmt.Errorf("error while inserting into availments collection %v", err.Error())
// 	}

// 	session.Set("availmentId", availmentsInsertRecord.InsertedID)
// 	session.Set("brandId", id)

// 	//if it's failure, then we just delete all the sessions and return
// 	if !record.Result {
// 		utils.SessionDeletion(session)
// 		return "Inserted Successfully", nil
// 	}
// 	message, err := SuccessInsertion(session)
// 	if err != nil {
// 		return "", err
// 	}

// 	return message, nil
// }
