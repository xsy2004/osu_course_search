package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"os"
)

//create table course
//(
//section_id           integer
//constraint section_id
//primary key,
//course_name          text,
//department           text,
//term                 integer,
//term_text            text,
//section_date         text,
//section_room         text,
//section_instructor   text,
//meeting_period       text,
//section_availability text
//);

var rdb *redis.Client
var conn *pgx.Conn

func init() {

	databaseUrl := "postgres://localhost:5432/postgres"
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       5,
	})
	var err error

	conn, err = pgx.Connect(context.Background(), databaseUrl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

}

type dbStruct struct {
	sectionId           int
	courseName          string
	department          string
	term                int
	termText            string
	sectionDate         string
	sectionRoom         string
	sectionInstructor   string
	meetingPeriod       string
	sectionAvailability string
}

func insertPsg(data dbStruct) error {
	_, err := conn.Exec(context.Background(), "INSERT INTO course(section_id, course_name, department, term, term_text, section_date, section_room, section_instructor, meeting_period, section_availability) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)", data.sectionId, data.courseName, data.department, data.term, data.termText, data.sectionDate, data.sectionRoom, data.sectionInstructor, data.meetingPeriod, data.sectionAvailability)
	return err
}

func updatePsgAvailability(sectionId int, sectionAvailability string) error {
	_, err := conn.Exec(context.Background(), "UPDATE course SET section_availability=$1 WHERE section_id=$2", sectionAvailability, sectionId)
	return err
}

func updatePsgSectionInstructor(sectionId int, sectionInstructor string) error {
	// 首先，获取当前的section_instructor值
	var currentInstructor string
	err := conn.QueryRow(context.Background(), "SELECT section_instructor FROM course WHERE section_id=$1", sectionId).Scan(&currentInstructor)
	if err != nil {
		// Handle error, for example if the section does not exist
		return err
	}

	// 比较当前instructor和传入的instructor是否相等
	if currentInstructor == sectionInstructor {
		// 如果相等，不需要更新，可以直接返回nil或者一个自定义的错误表示没有变化
		return nil
	}

	// 如果不相等，执行更新操作
	_, err = conn.Exec(context.Background(), "UPDATE course SET section_instructor = section_instructor || ',' || $1 WHERE section_id=$2", sectionInstructor, sectionId)
	return err
}

func checkSectionIdExist(sectionId int) bool {
	var columnNameCheck string
	err := conn.QueryRow(context.Background(),
		"SELECT course FROM course WHERE section_id = $1 ",
		sectionId).Scan(&columnNameCheck)

	if err == pgx.ErrNoRows {
		return false
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "Query failed: %v\n", err)
		os.Exit(1)
	} else {
		return true
	}
	return false
}

func updateRedis(key string, data string) bool {
	ctx := context.Background()
	err := rdb.SAdd(ctx, key, data).Err()

	if err != nil {
		fmt.Println(err)
		return false
	}

	return true
}

func getRedisSpecificExist(key string, value string) bool {
	ctx := context.Background()
	memberExists, err := rdb.SIsMember(ctx, key, value).Result()
	if err != nil {
		panic(err)
	}

	return memberExists

}

func getRedisItemLength(key string) int64 {
	ctx := context.Background()
	setLength, err := rdb.SCard(ctx, key).Result()
	if err != nil {
		panic(err)
	}
	return setLength
}
