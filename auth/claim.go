package auth

import (
	"crypto/rand"
	"math/big"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func CreateClaims(c *fiber.Ctx, uuid string, expTime time.Duration) jwt.MapClaims {
	bigInt := new(big.Int)
	uBit128 := "340282366920938463463374607431768211455"
	bigInt.SetString(uBit128, 10)
	rng, _ := rand.Int(rand.Reader, bigInt)
	return jwt.MapClaims{
		"sub": uuid,
		"aud": c.IP(),
		"jwi": rng,
		"exp": time.Now().Add(time.Duration(expTime)).Unix(),
	}
}
