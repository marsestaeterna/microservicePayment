package generateString

import(
    "math/rand"
)
type GenerateString struct {
    String string
}
var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890-")
var orderNumberSize int = 32

func (gen *GenerateString)RandStringRunes() {
    b := make([]rune, orderNumberSize)
    for i := range b {
      b[i] = letterRunes[rand.Intn(len(letterRunes))] 
    }
   gen.String = string(b)

}