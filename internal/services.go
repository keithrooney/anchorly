package internal

import (
	"errors"
	"os"
	"time"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

var secret []byte

func init() {
	if value, found := os.LookupEnv("ANCHORLY_TOKEN_KEY"); !found {
		panic("Expected environment variable to be configured.")
	} else {
		secret = []byte(value)
	}
}

type UserService interface {
	Create(user User) (User, error)
	GetById(id string) (User, error)
	GetByEmail(email string) (User, error)
	Exists(id string) bool
}

type repositoryUserService struct {
	userRepository UserRepository
}

func (s repositoryUserService) Create(user User) (User, error) {
	if err := validation.Validate(
		user.Username,
		validation.Required,
		validation.Length(4, 250),
	); err != nil {
		return User{}, errors.New("username is invalid")
	}
	if err := validation.Validate(
		user.Email,
		validation.Required,
		is.Email,
	); err != nil {
		return User{}, errors.New("email is invalid")
	}
	password := user.Password
	if err := validation.Validate(
		password,
		validation.Required,
		validation.Length(8, 500),
	); err != nil {
		return User{}, errors.New("password is invalid")
	}
	hp, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, errors.New("internal server error")
	}
	clone, err := s.userRepository.Create(User{
		Username: user.Username,
		Email:    user.Email,
		Password: string(hp),
	})
	if err != nil {
		return User{}, errors.New("internal server error")
	}
	return clone, nil
}

func (s repositoryUserService) GetById(id string) (User, error) {
	user, err := s.userRepository.GetById(id)
	if err != nil {
		return User{}, errors.New("object not found")
	}
	return user, nil
}

func (s repositoryUserService) GetByEmail(email string) (User, error) {
	user, err := s.userRepository.GetByEmail(email)
	if err != nil {
		return User{}, errors.New("object not found")
	}
	return user, nil
}

func (s repositoryUserService) Exists(id string) bool {
	_, err := s.GetById(id)
	return err == nil
}

func NewUserService() UserService {
	return repositoryUserService{
		userRepository: newUserRepository(),
	}
}

type LoginService interface {
	Login(user User) (Token, error)
}

func (s repositoryUserService) Login(other User) (Token, error) {
	user, err := s.GetByEmail(other.Email)
	if err != nil {
		return Token{}, errors.New("bad request")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(other.Password)); err != nil {
		return Token{}, errors.New("permission denied")
	}
	claims := jwt.MapClaims{
		"iss": "anchorly.com",
		"sub": user.Model.ID,
		"aud": user.Model.ID,
		"exp": time.Now().Add(time.Hour * 3).UnixMilli(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	value, err := token.SignedString(secret)
	if err != nil {
		return Token{}, errors.New("internal server error")
	}
	return Token{
		Value: value,
	}, nil
}

type AuthenticationService interface {
	Authenticate(token string)
}

func (s repositoryUserService) Authenticate(tokenString string) error {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("internal server error")
		}
		return secret, nil
	})
	if err != nil {
		return errors.New("internal server error")
	}
	if token.Valid {
		return nil
	} else {
		return errors.New("permission denied")
	}
}

type LinkService interface {
	Create(link Link) (Link, error)
	GetById(id string) (Link, error)
}

type repositoryLinkService struct {
	linkRepository LinkRepository
}

func (s repositoryLinkService) Create(link Link) (Link, error) {
	if err := validation.Validate(
		link.Title,
		validation.Required,
		validation.Length(4, 250),
	); err != nil {
		return Link{}, errors.New("title is invalid")
	}
	if err := validation.Validate(
		link.Href,
		validation.Required,
		is.URL,
	); err != nil {
		return Link{}, errors.New("href is invalid")
	}
	if err := validation.Validate(
		link.User.ID,
		validation.Required,
		is.UUID,
	); err != nil {
		return Link{}, errors.New("user is required")
	}
	clone, err := s.linkRepository.Create(link)
	if err != nil {
		return Link{}, errors.New("internal server error")
	}
	return clone, nil
}

func (s repositoryLinkService) GetById(id string) (Link, error) {
	link, err := s.linkRepository.GetById(id)
	if err != nil {
		return Link{}, errors.New("object not found")
	}
	return link, nil
}

func NewLinkService() LinkService {
	return repositoryLinkService{
		linkRepository: newLinkRepository(),
	}
}
