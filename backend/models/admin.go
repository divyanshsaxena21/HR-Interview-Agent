package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Admin struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"_id"`
	AdminID  string             `bson:"admin_id" json:"admin_id"`
	Password string             `bson:"password" json:"password"`
	Email    string             `bson:"email" json:"email"`
	Name     string             `bson:"name" json:"name"`
}

type AdminLoginRequest struct {
	AdminID  string `json:"admin_id" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AdminLoginResponse struct {
	Token string `json:"token"`
	Name  string `json:"name"`
}
