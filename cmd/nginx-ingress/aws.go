// +build aws

package main

import (
	"context"
	"encoding/base64"
	"errors"
	"log"
	"math/rand"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/marketplacemetering"
	"github.com/aws/aws-sdk-go-v2/service/marketplacemetering/types"

	"github.com/dgrijalva/jwt-go/v4"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-"

var (
	productCode   string
	pubKeyVersion int32 = 1
	pubKeyString  string
	nonce         string
)

func init() {
	rand.Seed(jwt.Now().UnixNano())
	nonce = RandStringBytesRmndr(32)

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Error loading AWS configuration: %v", err)
	}

	mpm := marketplacemetering.New(marketplacemetering.Options{Region: cfg.Region, Credentials: cfg.Credentials})

	out, err := mpm.RegisterUsage(context.TODO(), &marketplacemetering.RegisterUsageInput{ProductCode: &productCode, PublicKeyVersion: &pubKeyVersion, Nonce: &nonce})
	if err != nil {
		var notEnt *types.CustomerNotEntitledException
		if errors.As(err, &notEnt) {
			log.Fatalf("User not entitled, code: %v, message: %v, fault: %v", notEnt.ErrorCode(), notEnt.ErrorMessage(), notEnt.ErrorFault().String())
		}
		log.Fatal(err)

	}

	pk, err := base64.StdEncoding.DecodeString(pubKeyString)
	if err != nil {
		log.Fatal(err)
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pk)
	if err != nil {
		log.Fatal(err)
	}

	token, err := jwt.ParseWithClaims(*out.Signature, &Claims{}, jwt.KnownKeyfunc(jwt.SigningMethodPS256, pubKey))
	if err != nil {
		log.Fatal(err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		if claims.ProductCode == productCode && claims.PublicKeyVersion == pubKeyVersion && claims.Nonce == nonce {
			log.Println("AWS verification successful")
		} else {
			log.Fatal("The claims in the JWT token don't match the request")
		}
	} else {
		log.Fatal("Something is wrong with the JWT token")
	}
}

type Claims struct {
	ProductCode      string    `json:"productCode,omitempty"`
	PublicKeyVersion int32     `json:"publicKeyVersion,omitempty"`
	IssuedAt         *jwt.Time `json:"iat,omitempty"`
	Nonce            string    `json:"nonce,omitempty"`
}

func (c Claims) Valid(h *jwt.ValidationHelper) error {
	if c.Nonce == "" {
		return &jwt.InvalidClaimsError{Message: "The JWT token doesn't include the Nonce"}
	}
	if c.ProductCode == "" {
		return &jwt.InvalidClaimsError{Message: "The JWT token doesn't include the ProductCode"}
	}
	if c.PublicKeyVersion == 0 {
		return &jwt.InvalidClaimsError{Message: "The JWT token doesn't include the PublicKeyVersion"}
	}
	if h.Before(c.IssuedAt.Time) {
		return &jwt.InvalidClaimsError{Message: "The JWT token has a wrong creation time"}
	}

	return nil
}

func RandStringBytesRmndr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}
