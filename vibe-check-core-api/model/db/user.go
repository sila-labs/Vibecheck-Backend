package model

import "time"

// originally customer data type, updating to User name
type User struct {
	UserId      string    `json:"user_id" db:"user_id"`
	UserName    string    `json:"user_name" db:"user_name"`
	Password    string    `json:"password" db:"password"`
	Email       string    `json:"email" db:"email"`
	DateCreated time.Time `json:"date_created" db:"date_created"`
	DateUpdated time.Time `json:"date_updated" db:"date_updated"`
	Phone       string    `json:"phone" db:"phone"`
	Photo       bool      `json:"photo" db:"photo"`
	FirstName   string    `json:"first_name" db:"first_name"`
	LastName    string    `json:"last_name" db:"last_name"`
	IsDeleted   bool      `json:"is_deleted" db:"is_deleted"`
}

type UserFollower struct {
	UserId          string `json:"user_id" db:"user_id"`
	UserIdFollowing string `json:"user_id_following" db:"user_id_following"`
}

type UserRequest struct {
	UserId string `json:"user_id" db:"user_id"`
}

type FFCounts struct {
	FollowerCount  int `json:"follower_count" db:"follower_count"`
	FollowingCount int `json:"following_count" db:"following_count"`
}

// current Users schema
// CREATE TABLE `users` (
// 	`user_id` varchar(36) NOT NULL,
// 	`user_name` varchar(20) NOT NULL,
// 	`password` char(60) NOT NULL,
// 	`email` varchar(100) DEFAULT NULL,
// 	`date_created` datetime(3) DEFAULT current_timestamp(3),
// 	`date_updated` datetime(3) DEFAULT current_timestamp(3),
// 	`phone` varchar(15) NOT NULL,
// 	`photo` BOOLEAN NOT NULL,
// 	`first_name` VARCHAR(255) NOT NULL,
// 	`last_name` VARCHAR(255),
// 	PRIMARY KEY (`user_id`)
//   ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

// type Customer struct {
// 	Id          string    `json:"id" db:"id"`
// 	Email       string    `json:"email" db:"email"`
// 	Password    string    `json:"password" db:"password"`
// 	Username    string    `json:"username" db:"username"`
// 	Phone       string    `json:"phone" db:"phone"`
// 	DateCreated time.Time `json:"date_created" db:"date_created"`
// 	DateUpdated time.Time `json:"date_updated" db:"date_updated"`
// }
