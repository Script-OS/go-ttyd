package ttyd

type AuthConfig struct {
	//Certificate *string // sha256 string of password
	//Salt        *string // salt string , random generated
	Certificate *string // password
}

func NewAuth(password *string) *AuthConfig {
	//salt := strconv.FormatInt(rand.Int63(), 10)
	//return &AuthConfig{Certificate: nil, Salt: &salt}
	if *password == "" {
		return nil
	}
	return &AuthConfig{Certificate: password}
}

//func (config *AuthConfig) MakeSha256(password string) error {
//
//	hash1 := sha256.New()
//	_, err := io.WriteString(hash1, *config.Salt+password)
//	if err != nil {
//		log.Panicln(err.Error())
//		return err
//	}
//	hash2 := sha256.New()
//	_, err = io.WriteString(hash2, *config.Salt+fmt.Sprintf("%02x", hash1.Sum(nil)))
//	if err != nil {
//		log.Panicln(err.Error())
//		return err
//	}
//	*config.Certificate = fmt.Sprintf("%02x", hash2.Sum(nil))
//	return nil
//}

func (config *AuthConfig) Check(password string) bool {
	if *config.Certificate == password {
		return true
	} else {
		return false
	}
}
