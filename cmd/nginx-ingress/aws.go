// +build aws

package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"math/big"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/marketplacemetering"
	"github.com/aws/aws-sdk-go-v2/service/marketplacemetering/types"

	"github.com/dgrijalva/jwt-go/v4"
)

var (
	productCode   string
	pubKeyVersion int32 = 1
	pubKeyString  string
	nonce         string
)

func init() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	nonce, err := generateRandomString(255)
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("error loading AWS configuration: %v", err)
	}

	mpm := marketplacemetering.New(marketplacemetering.Options{Region: cfg.Region, Credentials: cfg.Credentials})

	out, err := mpm.RegisterUsage(ctx, &marketplacemetering.RegisterUsageInput{ProductCode: &productCode, PublicKeyVersion: &pubKeyVersion, Nonce: &nonce})
	if err != nil {
		var notEnt *types.CustomerNotEntitledException
		if errors.As(err, &notEnt) {
			log.Fatalf("user not entitled, code: %v, message: %v, fault: %v", notEnt.ErrorCode(), notEnt.ErrorMessage(), notEnt.ErrorFault().String())
		}
		log.Fatal(err)

	}

	pk, err := base64.StdEncoding.DecodeString(pubKeyString)
	if err != nil {
		log.Fatalf("error decoding Public Key string: %v", err)
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pk)
	if err != nil {
		log.Fatalf("error parsing Public Key: %v", err)
	}

	token, err := jwt.ParseWithClaims(*out.Signature, &claims{}, jwt.KnownKeyfunc(jwt.SigningMethodPS256, pubKey))
	if err != nil {
		log.Fatalf("error parsing the JWT token: %v", err)
	}

	if claims, ok := token.Claims.(*claims); ok && token.Valid {
		if claims.ProductCode == productCode && claims.PublicKeyVersion == pubKeyVersion && claims.Nonce == nonce {
			log.Println("AWS verification successful")
		} else {
			log.Fatal("the claims in the JWT token don't match the request")
		}
	} else {
		log.Fatal("something is wrong with the JWT token")
	}
}

type claims struct {
	ProductCode      string    `json:"productCode,omitempty"`
	PublicKeyVersion int32     `json:"publicKeyVersion,omitempty"`
	IssuedAt         *jwt.Time `json:"iat,omitempty"`
	Nonce            string    `json:"nonce,omitempty"`
}

func (c claims) Valid(h *jwt.ValidationHelper) error {
	if c.Nonce == "" {
		return &jwt.InvalidClaimsError{Message: "the JWT token doesn't include the Nonce"}
	}
	if c.ProductCode == "" {
		return &jwt.InvalidClaimsError{Message: "the JWT token doesn't include the ProductCode"}
	}
	if c.PublicKeyVersion == 0 {
		return &jwt.InvalidClaimsError{Message: "the JWT token doesn't include the PublicKeyVersion"}
	}
	if err := h.ValidateNotBefore(c.IssuedAt); err != nil {
		return err
	}

	return nil
}

func generateRandomString(n int) (string, error) {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"
	ret := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		ret[i] = letters[num.Int64()]
	}

	return string(ret), nil
}
