package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

//JWTSCRETKEY ...
var JWTSCRETKEY = []byte("TyCjiKGI9AhFstuVzSmwsXKyIbqR5iQAZqfd06vTJY5pJA5skClewZVOTtmw88KY")

//NewJWTTokenWithClaims ...
func NewJWTTokenWithClaims(claims *UserClaim, scretKey []byte) string {

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// fmt.Println(token)
	signedToken, _ := token.SignedString(scretKey)
	// fmt.Println(signedToken)
	return signedToken
}

//ParseJwtTokenWithClaims ...
func ParseJwtTokenWithClaims(tokenString string, scretKey []byte) (*UserClaim, error) {
	userClaim := new(UserClaim)
	_, err := jwt.ParseWithClaims(tokenString, userClaim, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			//TODO
			return nil, fmt.Errorf("Unexpected user claims signing method: %v", token.Header["alg"])
		}
		if err := token.Claims.Valid(); err != nil {
			//TODO
			return nil, fmt.Errorf("Unexpected user claims: %v", err)
		}
		return scretKey, nil
	})
	if err != nil {
		return userClaim, err
	}
	return userClaim, nil
}

func main() {
	expireToken := time.Now().Add(time.Second * 60 * 10).Unix()
	claims := &UserClaim{
		ID:   1,
		Name: "张张",
		Age:  27,
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: expireToken,
			Issuer:    "next-stage.com.cn",
		},
	}
	token := NewJWTTokenWithClaims(claims, JWTSCRETKEY)
	fmt.Println(token)

	db, err := sqlx.Open("mysql", "root:root@tcp(127.0.0.1:3306)/test")
	if err != nil {
		log.Println("mysql connect failed")
		return
	}
	defer db.Close()
	mux := http.NewServeMux()
	mux.Handle("/v1/getUser", JWTAuth(getUserHandler(db)))
	mux.Handle("/v1/listUser", JWTAuth(listUserHandler(db)))
	mux.Handle("/v1/deleteUser", JWTAuth(deleteUserHandler(db)))
	mux.Handle("/v1/addUser", JWTAuth(addUserHandler(db)))
	mux.Handle("/v1/updateUser", JWTAuth(updateUserHandler(db)))
	http.ListenAndServe(":8091", mux)

}

//GenericResults ...
type GenericResults struct {
	Status  int32       `json:"Status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func connectDb() *sqlx.DB {
	db, err := sqlx.Open("mysql", "root:root@tcp(127.0.0.1:3306)/test")
	if err != nil {
		log.Println("mysql connect failed")
		return nil
	}
	return db
}

//User ...
type User struct {
	ID   int64  `json:"id,string" db:"id"`
	Name string `json:"name" db:"name"`
	Age  int64  `json:"age,string" db:"age"`
}

//UserClaim ...
type UserClaim struct {
	ID   int64
	Name string
	Age  int64
	jwt.StandardClaims
}

func getUserHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		idParseForm := r.Form["id"][0]
		id, err := strconv.ParseInt(idParseForm, 10, 64)
		result := GenericResults{
			Status: http.StatusOK,
		}
		if err != nil {
			result.Status = http.StatusInternalServerError
			result.Message = fmt.Sprintf("参数错误,%v", id)
			jsons, _ := json.Marshal(result)
			w.Write(jsons)
			return
		}
		result.Data, err = getUser(db, id)

		if err != nil {
			result.Status = http.StatusInternalServerError
			result.Message = fmt.Sprintf("查询用户失败,%v", err)
		}
		jsons, _ := json.Marshal(result)
		w.Write(jsons)
	}
}

func listUserHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// r.Close = true
		// w.Header().Set("Connection", "close")
		result := GenericResults{
			Status: http.StatusOK,
		}
		results, err := listUser(db)

		if err != nil {
			result.Status = http.StatusInternalServerError
			result.Message = fmt.Sprintf("查询用户失败,%v", err)
			jsons, _ := json.Marshal(result)
			w.Write(jsons)
		}
		result.Data = results
		jsons, _ := json.Marshal(result)
		w.Write(jsons)
	}
}

func deleteUserHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		id := r.Form["id"][0]
		idint, err := strconv.ParseInt(id, 10, 64)
		result := GenericResults{
			Status: http.StatusOK,
		}
		if err != nil {
			result.Status = http.StatusInternalServerError
			result.Message = fmt.Sprintf("参数错误,%v", id)
			jsons, _ := json.Marshal(result)
			w.Write(jsons)
			return
		}
		rowsAffected, err := deleteUser(db, idint)
		if err != nil {
			result.Status = http.StatusInternalServerError
			result.Message = fmt.Sprintf("删除用户失败,%v", err)
			jsons, _ := json.Marshal(result)
			w.Write(jsons)
			return
		}
		if rowsAffected < 1 {
			result.Message = fmt.Sprintf("用户id不存在,%v", err)
		} else {
			result.Message = fmt.Sprintf("删除成功")
		}
		jsons, _ := json.Marshal(result)
		w.Write(jsons)
	}
}

func addUserHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		name := r.Form["name"][0]
		ageParseForm := r.Form["age"][0]
		age, err := strconv.ParseInt(ageParseForm, 10, 64)
		result := GenericResults{
			Status: http.StatusOK,
		}
		if err != nil {
			result.Status = http.StatusInternalServerError
			result.Message = fmt.Sprintf("参数错误,%v", age)
			jsons, _ := json.Marshal(result)
			w.Write(jsons)
			return
		}
		user := User{}
		user.Name = name
		user.Age = age
		err = addUser(db, &user)
		if err != nil {
			result.Status = http.StatusInternalServerError
			result.Message = fmt.Sprintf("添加用户失败,%v", err)
			jsons, _ := json.Marshal(result)
			w.Write(jsons)
			return
		}
		result.Message = fmt.Sprintf("新增用户成功")
		jsons, _ := json.Marshal(result)
		w.Write(jsons)
		return
	}
}

func updateUserHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		idParseForm := r.Form["id"][0]
		name := r.Form["name"][0]
		ageParseForm := r.Form["age"][0]
		age, err := strconv.ParseInt(ageParseForm, 10, 64)
		id, err1 := strconv.ParseInt(idParseForm, 10, 64)
		result := GenericResults{
			Status: http.StatusOK,
		}
		if err != nil || err1 != nil {
			result.Status = http.StatusInternalServerError
			result.Message = fmt.Sprintf("参数错误,%v", age)
			jsons, _ := json.Marshal(result)
			w.Write(jsons)
			return
		}
		user := User{}
		user.Name = name
		user.Age = age
		user.ID = id
		row, err := updateUser(db, &user)
		if err != nil {
			result.Status = http.StatusInternalServerError
			result.Message = fmt.Sprintf("用户修改失败,%v", err)
			jsons, _ := json.Marshal(result)
			w.Write(jsons)
			return
		}
		if row < 1 {
			result.Status = http.StatusInternalServerError
			result.Message = fmt.Sprintf("用户id不存在")
			jsons, _ := json.Marshal(result)
			w.Write(jsons)
			return
		}
		result.Message = fmt.Sprintf("修改成功")
		jsons, _ := json.Marshal(result)
		w.Write(jsons)
	}
}

//查询用户
func getUser(db *sqlx.DB, id int64) (*User, error) {
	result := new(User)
	err := db.Get(result, "SELECT id, name ,age FROM user WHERE id = ?", id)
	switch err {
	case sql.ErrNoRows:
		return nil, nil
	case nil:
		return result, nil
	default:
		return nil, err
	}
}

func listUser(db *sqlx.DB) (*[]User, error) {
	results := []User{}
	err := db.Select(&results, "SELECT id, name ,age FROM user LIMIT 1000")
	if err != nil {
		return nil, err
	}
	return &results, nil
}

//删除用户
func deleteUser(db *sqlx.DB, id int64) (int64, error) {
	result, err := db.Exec("DELETE FROM user WHERE id = ?", id)
	if err != nil {
		return 0, err
	}
	rowsAffected, _ := result.RowsAffected()
	return rowsAffected, nil
}

//新增用户
func addUser(db *sqlx.DB, user *User) error {
	_, err := db.Exec("INSERT INTO user values(null,?,?)", user.Name, user.Age)
	return err
}

//修改用户
func updateUser(db *sqlx.DB, user *User) (int64, error) {
	result, err := db.Exec("UPDATE user SET name=?,age=? WHERE id=?", user.Name, user.Age, user.ID)
	if err != nil {
		return 0, err
	}
	row, _ := result.RowsAffected()
	return row, nil
}

//JWTAuth token ...
func JWTAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Close = true
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Connection", "close")
		authorizationHeader := r.Header.Get("Authorization")
		fmt.Println(authorizationHeader)
		result := GenericResults{
			Status: http.StatusOK,
		}
		if len(authorizationHeader) == 0 {
			result.Status = http.StatusUnauthorized
			result.Message = "authorization required."
			jsons, _ := json.Marshal(result)
			w.Write(jsons)
			return
		}
		if len(authorizationHeader) >= 4 && authorizationHeader[0:4] == "JWT " {
			var token *UserClaim
			token, err := ParseJwtTokenWithClaims(authorizationHeader[4:], JWTSCRETKEY)
			if err != nil {
				result.Status = http.StatusUnauthorized
				result.Message = fmt.Sprintf("authorization required. %s", err.Error())
				jsons, _ := json.Marshal(result)
				w.Write(jsons)
				fmt.Println(token)
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		result.Status = http.StatusUnauthorized
		result.Message = fmt.Sprintf("authorization validation failed")
		jsons, _ := json.Marshal(result)
		w.Write(jsons)
	})
}
