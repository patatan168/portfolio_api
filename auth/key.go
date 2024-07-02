package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"math/big"
)

func CreatePrivateKey(key string) *ecdsa.PrivateKey {
	privateKey := new(ecdsa.PrivateKey)
	keyBigInt := new(big.Int)
	keyBigInt.SetString(key, 10)
	curve := elliptic.P256()
	privateKey.PublicKey.Curve = curve
	privateKey.D = keyBigInt
	privateKey.PublicKey.X, privateKey.PublicKey.Y = curve.ScalarBaseMult(keyBigInt.Bytes())
	return privateKey
}
