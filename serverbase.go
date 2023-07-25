// DynamoDBを使用しているため、このプログラムのみでは動きません。
// インターネット上にデプロイしていますのでクライアントプログラムからそちらにアクセスしていただければと存じます。
package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// ユーザデータ
type User struct {
	ID   int
	Name string
	Lv   int
	Exp  int
}

// タスク内容　※数字はDBにおけるリストの何番目かを指す
type Task struct {
	ID        int    //userID
	Tasknum   int    //何番目のタスクか
	Task      string //タスク内容：0
	Taskcheck string //完了未完了(完了日)：1
	Regdate   string //登録日：2
	Deadline  string //期限：3
	Per       int    //進行度：4
	Important int    //重要度：5
	EnemyID   int    //敵ID：6
	Memo      string //メモ：7
}

// 友人登録・解除用
type Friend struct {
	ID  int
	FID int
}

// 友人戦績用
type Friendscore struct {
	ID       int
	Name     string
	Lv       int
	Exp      int
	Complete int //完了タスク数
	Total    int //完了+期限前未完了数
}

// main
func main() {
	api := rest.NewApi()
	api.Use(rest.DefaultDevStack...)

	router, handleerr := rest.MakeRouter(
		rest.Post("/reguser", RegUser),
		rest.Post("/getuser", GetUser),
		rest.Post("/chuser", ChangeUser),
		rest.Post("/taketask", TakeTask),
		rest.Post("/posttask", RegTask),
		rest.Post("/chtask", ChangeTask),
		rest.Post("/deltask", DeleteTask),
		rest.Post("/regfri", RegFriend),
		rest.Post("/chfri", CheckFriend),
		rest.Post("/delfri", DeleteFriend),
	)
	if handleerr != nil {
		log.Fatal(handleerr)
	}
	api.SetApp(router)
	log.Printf("Server On")
	log.Fatal(http.ListenAndServe(":8264", api.MakeHandler()))

}

// ユーザ情報の新規登録
func RegUser(w rest.ResponseWriter, r *rest.Request) {
	//まずユーザ名をクライアントから受け取る
	cdata := User{}
	recerr := r.DecodeJsonPayload(&cdata)
	if recerr != nil {
		rest.Error(w, recerr.Error(), http.StatusInternalServerError)
		return //受信してないので終わり
	}
	uname := cdata.Name

	nowsession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("ap-northeast-1"), //DBのあるリージョン
	}))

	db := dynamodb.New(nowsession, aws.NewConfig().WithRegion("ap-northeast-1"))

	//ユーザIDを決める
	var init int = 0 //
	var check string = ""
	for {
		tmpparam := &dynamodb.GetItemInput{
			TableName: aws.String("Userdata"),
			Key: map[string]*dynamodb.AttributeValue{
				"ID": {
					N: aws.String(strconv.Itoa(init)),
				},
			},
			AttributesToGet: []*string{
				aws.String("ID"),
			},
			ConsistentRead: aws.Bool(true),

			ReturnConsumedCapacity: aws.String("NONE"),
		}
		tmpdata, _ := db.GetItem(tmpparam) //ない場合はtmpdataが空欄「{}」になる
		check = tmpdata.String()           //返答をstring化
		if check == "{\n\n}" {             //データがない場合「{\n\n}」はそのinitが登録IDとなる
			break
		}
		init += 1
	}
	check = strconv.Itoa(init) //文字列化

	//ユーザ情報を登録
	param := &dynamodb.PutItemInput{
		TableName: aws.String("Userdata"),
		Item: map[string]*dynamodb.AttributeValue{
			"ID": {
				N: aws.String(check),
			},
			"Name": {
				S: aws.String(uname),
			},
			"Lv": {
				N: aws.String("0"),
			},
			"exp": {
				N: aws.String("0"),
			},
		},
	}
	_, posterr := db.PutItem(param) //登録
	if posterr != nil {
		fmt.Println(posterr.Error())
	}

	//ユーザIDを返す
	var senddata *User
	senddata = &User{}
	senddata.ID, _ = strconv.Atoi(check)
	senddata.Name = uname
	senddata.Lv = 0
	senddata.Exp = 0
	w.WriteJson(senddata)
}

// ユーザ情報の取得
func GetUser(w rest.ResponseWriter, r *rest.Request) {
	//まずユーザIDをクライアントから受け取る
	cdata := User{}
	recerr := r.DecodeJsonPayload(&cdata)
	if recerr != nil {
		rest.Error(w, recerr.Error(), http.StatusInternalServerError)
		return //受信してないので終わり
	}
	uid := strconv.Itoa(cdata.ID)

	nowsession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("ap-northeast-1"), //DBのあるリージョン
	}))

	db := dynamodb.New(nowsession, aws.NewConfig().WithRegion("ap-northeast-1"))

	//ユーザ情報の取得
	param := &dynamodb.GetItemInput{
		TableName: aws.String("Userdata"),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				N: aws.String(uid),
			},
		},
		AttributesToGet: []*string{
			aws.String("ID"),
			aws.String("Name"),
			aws.String("Lv"),
			aws.String("exp"),
		},
		ConsistentRead:         aws.Bool(true),
		ReturnConsumedCapacity: aws.String("NONE"),
	}

	data, geterr := db.GetItem(param)
	var senddata *User
	if geterr != nil {
		fmt.Println(geterr.Error())
		return
	} else {
		senddata = &User{}
		senddata.ID, _ = strconv.Atoi(*data.Item["ID"].N)
		senddata.Name = *data.Item["Name"].S
		senddata.Lv, _ = strconv.Atoi(*data.Item["Lv"].N)
		senddata.Exp, _ = strconv.Atoi(*data.Item["exp"].N)
	}

	//クライアントに返す
	w.WriteJson(senddata)
}

// ユーザ情報変更・レベル上昇・経験値上昇
func ChangeUser(w rest.ResponseWriter, r *rest.Request) {
	//まずユーザID・ユーザ名・Lv・expをクライアントから受け取る
	cdata := User{}
	recerr := r.DecodeJsonPayload(&cdata)
	if recerr != nil {
		rest.Error(w, recerr.Error(), http.StatusInternalServerError)
		return //受信してないので終わり
	}
	uid := strconv.Itoa(cdata.ID)
	uname := cdata.Name
	ulv := strconv.Itoa(cdata.Lv)
	uexp := strconv.Itoa(cdata.Exp)

	nowsession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("ap-northeast-1"), //DBのあるリージョン
	}))

	db := dynamodb.New(nowsession, aws.NewConfig().WithRegion("ap-northeast-1"))

	//ユーザ情報を変更
	param := &dynamodb.UpdateItemInput{
		TableName: aws.String("Userdata"),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				N: aws.String(uid), // 既存キー名を指定
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#Name": aws.String("Name"),
			"#Lv":   aws.String("Lv"),
			"#exp":  aws.String("exp"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":Name": {
				S: aws.String(uname),
			},
			":Lv": {
				N: aws.String(ulv),
			},
			":exp": {
				N: aws.String(uexp),
			},
		},
		ReturnConsumedCapacity: aws.String("NONE"),
		UpdateExpression:       aws.String("set #Name = :Name,#Lv=:Lv,#exp=:exp"), //変更の定義
	}
	_, changeerr := db.UpdateItem(param)
	if changeerr != nil {
		fmt.Println(changeerr.Error())
		return
	}
}

// タスク取得(友人戦績時のタスク詳細もこれで行う)
func TakeTask(w rest.ResponseWriter, r *rest.Request) {
	//まずユーザIDをクライアントから受け取る
	cdata := User{}
	recerr := r.DecodeJsonPayload(&cdata)
	if recerr != nil {
		rest.Error(w, recerr.Error(), http.StatusInternalServerError)
		return //受信してないので終わり
	}
	uid := strconv.Itoa(cdata.ID)

	nowsession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("ap-northeast-1"), //DBのあるリージョン
	}))

	db := dynamodb.New(nowsession, aws.NewConfig().WithRegion("ap-northeast-1"))

	//タスク情報の取得
	param := &dynamodb.GetItemInput{
		TableName: aws.String("Userdata"),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				N: aws.String(uid),
			},
		},
		AttributesToGet: []*string{
			aws.String("tasklist"),
		},
		ConsistentRead:         aws.Bool(true),
		ReturnConsumedCapacity: aws.String("NONE"),
	}
	data, geterr := db.GetItem(param)
	var tmpdata *Task
	//タスク数送信→タスク0送信→タスク1送信…というようにしている
	if geterr != nil {
		fmt.Println(geterr.Error())
		return
	} else {
		w.WriteJson(len(data.Item["tasklist"].L)) //まずタスク数を送信
		for i := 0; i < len(data.Item["tasklist"].L); i++ {
			tmpdata = &Task{}
			tmpdata.ID, _ = strconv.Atoi(uid)
			tmpdata.Tasknum = i
			tmpdata.Task = *data.Item["tasklist"].L[i].L[0].S
			tmpdata.Taskcheck = *data.Item["tasklist"].L[i].L[1].S
			tmpdata.Regdate = *data.Item["tasklist"].L[i].L[2].S
			tmpdata.Deadline = *data.Item["tasklist"].L[i].L[3].S
			tmpdata.Per, _ = strconv.Atoi(*data.Item["tasklist"].L[i].L[4].N)
			tmpdata.Important, _ = strconv.Atoi(*data.Item["tasklist"].L[i].L[5].N)
			tmpdata.EnemyID, _ = strconv.Atoi(*data.Item["tasklist"].L[i].L[6].N)
			tmpdata.Memo = *data.Item["tasklist"].L[i].L[7].S
			w.WriteJson(tmpdata) //タスクデータ送信
		}
	}
}

// タスク新規登録
func RegTask(w rest.ResponseWriter, r *rest.Request) {
	//まずTask(ユーザID＋タスク情報(Task,Regdate,Deadline,Important,EnemyID,Memo))をクライアントから受け取る
	//日付関係は「yyyymmdd」で送られるので「yyyymmdd」→「yyyy/mm/dd」の処理をDB登録前にする
	cdata := Task{}
	recerr := r.DecodeJsonPayload(&cdata)
	if recerr != nil {
		rest.Error(w, recerr.Error(), http.StatusInternalServerError)
		return //受信してないので終わり
	}
	uid := strconv.Itoa(cdata.ID)
	task := cdata.Task
	regdate := cdata.Regdate
	regdate = regdate[0:4] + "/" + regdate[4:6] + "/" + regdate[6:8]
	deadline := cdata.Deadline
	deadline = deadline[0:4] + "/" + deadline[4:6] + "/" + deadline[6:8]
	important := strconv.Itoa(cdata.Important)
	enemyid := strconv.Itoa(cdata.EnemyID)
	memo := cdata.Memo

	nowsession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("ap-northeast-1"), //DBのあるリージョン
	}))

	db := dynamodb.New(nowsession, aws.NewConfig().WithRegion("ap-northeast-1"))

	//まずタスクリストの中身の各タスクの中身を作る
	var one []*dynamodb.AttributeValue
	taskdata := &dynamodb.AttributeValue{
		S: aws.String(task),
	}
	checkdata := &dynamodb.AttributeValue{
		S: aws.String("none"),
	}
	regdata := &dynamodb.AttributeValue{
		S: aws.String(regdate),
	}
	deaddata := &dynamodb.AttributeValue{
		S: aws.String(deadline),
	}
	perdata := &dynamodb.AttributeValue{
		N: aws.String("0"),
	}
	impdata := &dynamodb.AttributeValue{
		N: aws.String(important),
	}
	enmdata := &dynamodb.AttributeValue{
		N: aws.String(enemyid),
	}
	memodata := &dynamodb.AttributeValue{
		S: aws.String(memo),
	}
	one = append(one, taskdata, checkdata, regdata, deaddata, perdata, impdata, enmdata, memodata)

	//各タスクを作る
	tasklistdata := &dynamodb.AttributeValue{
		L: one,
	}

	//既にリストがあれば持ってくる
	param := &dynamodb.GetItemInput{
		TableName: aws.String("Userdata"),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				N: aws.String(uid),
			},
		},
		AttributesToGet: []*string{
			aws.String("tasklist"),
		},
		ConsistentRead:         aws.Bool(true),
		ReturnConsumedCapacity: aws.String("NONE"),
	}
	data, geterr := db.GetItem(param)
	if geterr != nil {
		fmt.Println(geterr.Error())
		return
	}

	var list []*dynamodb.AttributeValue
	if data.Item["tasklist"] == nil {
		//まだタスクリストがない
		list = append(list, tasklistdata)
	} else {
		//タスクリストあり
		list = append(data.Item["tasklist"].L, tasklistdata)
	}

	param2 := &dynamodb.UpdateItemInput{
		TableName: aws.String("Userdata"),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				N: aws.String(uid), // 既存キー名を指定
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#tasklist": aws.String("tasklist"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":tasklist": {
				L: list,
			},
		},
		ReturnConsumedCapacity: aws.String("NONE"),
		UpdateExpression:       aws.String("set #tasklist = :tasklist"), ////変更の定義
	}
	_, posterr := db.UpdateItem(param2) //登録
	if posterr != nil {
		fmt.Println(posterr.Error())
	}
}

// タスク変更・タスク進捗更新・タスククリア
func ChangeTask(w rest.ResponseWriter, r *rest.Request) {
	//まずユーザID＋DBの何番目のタスクか+タスク変更後情報(Task,Taskcheck,Deadline,Important,EnemyID,Memo)
	cdata := Task{}
	recerr := r.DecodeJsonPayload(&cdata)
	if recerr != nil {
		rest.Error(w, recerr.Error(), http.StatusInternalServerError)
		return //受信してないので終わり
	}
	uid := strconv.Itoa(cdata.ID)
	tasknum := cdata.Tasknum
	newtask := cdata.Task
	taskcheck := cdata.Taskcheck
	if taskcheck != "none" {
		taskcheck = taskcheck[0:4] + "/" + taskcheck[4:6] + "/" + taskcheck[6:8]
	}
	deadline := cdata.Deadline
	deadline = deadline[0:4] + "/" + deadline[4:6] + "/" + deadline[6:8]
	per := strconv.Itoa(cdata.Per)
	important := strconv.Itoa(cdata.Important)
	enemyid := strconv.Itoa(cdata.EnemyID)
	memo := cdata.Memo

	nowsession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("ap-northeast-1"), //DBのあるリージョン
	}))

	db := dynamodb.New(nowsession, aws.NewConfig().WithRegion("ap-northeast-1"))

	//タスクリストの取得
	param := &dynamodb.GetItemInput{
		TableName: aws.String("Userdata"),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				N: aws.String(uid),
			},
		},
		AttributesToGet: []*string{
			aws.String("tasklist"),
		},
		ConsistentRead:         aws.Bool(true),
		ReturnConsumedCapacity: aws.String("NONE"),
	}
	data, geterr := db.GetItem(param)
	if geterr != nil {
		fmt.Println(geterr.Error())
		return
	}

	//タスク情報変更
	*data.Item["tasklist"].L[tasknum].L[0].S = newtask
	*data.Item["tasklist"].L[tasknum].L[1].S = taskcheck
	*data.Item["tasklist"].L[tasknum].L[3].S = deadline
	*data.Item["tasklist"].L[tasknum].L[4].N = per
	*data.Item["tasklist"].L[tasknum].L[5].N = important
	*data.Item["tasklist"].L[tasknum].L[6].N = enemyid
	*data.Item["tasklist"].L[tasknum].L[7].S = memo

	//更新
	param2 := &dynamodb.UpdateItemInput{
		TableName: aws.String("Userdata"),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				N: aws.String(uid), // 既存キー名を指定
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#tasklist": aws.String("tasklist"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":tasklist": {
				L: data.Item["tasklist"].L,
			},
		},
		ReturnConsumedCapacity: aws.String("NONE"),
		UpdateExpression:       aws.String("set #tasklist = :tasklist"), ////変更の定義
	}
	_, posterr := db.UpdateItem(param2)
	if posterr != nil {
		fmt.Println(posterr.Error())
	}
}

// タスク削除
func DeleteTask(w rest.ResponseWriter, r *rest.Request) {
	//まずユーザID＋DBの何番目のタスクかをクライアントから受け取る
	cdata := Task{}
	recerr := r.DecodeJsonPayload(&cdata)
	if recerr != nil {
		rest.Error(w, recerr.Error(), http.StatusInternalServerError)
		return //受信してないので終わり
	}
	uid := strconv.Itoa(cdata.ID)
	tasknum := cdata.Tasknum

	nowsession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("ap-northeast-1"), //DBのあるリージョン
	}))

	db := dynamodb.New(nowsession, aws.NewConfig().WithRegion("ap-northeast-1"))
	//タスクリストの取得
	param := &dynamodb.GetItemInput{
		TableName: aws.String("Userdata"),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				N: aws.String(uid),
			},
		},
		AttributesToGet: []*string{
			aws.String("tasklist"),
		},
		ConsistentRead:         aws.Bool(true),
		ReturnConsumedCapacity: aws.String("NONE"),
	}
	data, geterr := db.GetItem(param)
	if geterr != nil {
		fmt.Println(geterr.Error())
		return
	}

	//更新処理を使って削除する(ユーザデータ全体ではないため)
	//tasknum番目のタスクを消したリストに更新
	list := append(data.Item["tasklist"].L[:tasknum], data.Item["tasklist"].L[tasknum+1:]...)

	//更新処理
	param2 := &dynamodb.UpdateItemInput{
		TableName: aws.String("Userdata"),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				N: aws.String(uid), // 既存キー名を指定
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#tasklist": aws.String("tasklist"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":tasklist": {
				L: list,
			},
		},
		ReturnConsumedCapacity: aws.String("NONE"),
		UpdateExpression:       aws.String("set #tasklist = :tasklist"), ////変更の定義
	}
	_, posterr := db.UpdateItem(param2)
	if posterr != nil {
		fmt.Println(posterr.Error())
	}

}

// 友達登録
func RegFriend(w rest.ResponseWriter, r *rest.Request) {
	//まずユーザID+登録する友人のユーザIDを取得
	cdata := Friend{}
	recerr := r.DecodeJsonPayload(&cdata)
	if recerr != nil {
		rest.Error(w, recerr.Error(), http.StatusInternalServerError)
		return //受信してないので終わり
	}
	uid := strconv.Itoa(cdata.ID)
	fuid := strconv.Itoa(cdata.FID)

	nowsession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("ap-northeast-1"), //DBのあるリージョン
	}))

	db := dynamodb.New(nowsession, aws.NewConfig().WithRegion("ap-northeast-1"))

	//既にフレンドリストがあれば持ってくる
	param := &dynamodb.GetItemInput{
		TableName: aws.String("Userdata"),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				N: aws.String(uid),
			},
		},
		AttributesToGet: []*string{
			aws.String("friendlist"),
		},
		ConsistentRead:         aws.Bool(true),
		ReturnConsumedCapacity: aws.String("NONE"),
	}
	data, geterr := db.GetItem(param)
	if geterr != nil {
		fmt.Println(geterr.Error())
		return
	}

	//リストの中身を作る
	fuiddata := &dynamodb.AttributeValue{
		N: aws.String(fuid),
	}
	var list []*dynamodb.AttributeValue
	if data.Item["friendlist"] == nil {
		//まだフレンドリストがない
		list = append(list, fuiddata)
	} else {
		//フレンドリストあり
		//まず重複がないか見る
		checker := false
		for _, value := range data.Item["friendlist"].L {
			if *value.N == fuid {
				checker = true
				break
			}
		}
		//重複しているか否かで追加処理するかを検討
		if !checker {
			list = append(data.Item["friendlist"].L, fuiddata)
		} else {
			list = data.Item["friendlist"].L
		}
	}

	param2 := &dynamodb.UpdateItemInput{
		TableName: aws.String("Userdata"),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				N: aws.String(uid), // 既存キー名を指定
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#friendlist": aws.String("friendlist"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":friendlist": {
				L: list,
			},
		},
		ReturnConsumedCapacity: aws.String("NONE"),
		UpdateExpression:       aws.String("set #friendlist = :friendlist"), ////変更の定義
	}
	_, posterr := db.UpdateItem(param2) //登録
	if posterr != nil {
		fmt.Println(posterr.Error())
	}

}

// 友達解除
func DeleteFriend(w rest.ResponseWriter, r *rest.Request) {
	//まずユーザID+削除する友人のユーザIDを取得
	cdata := Friend{}
	recerr := r.DecodeJsonPayload(&cdata)
	if recerr != nil {
		rest.Error(w, recerr.Error(), http.StatusInternalServerError)
		return //受信してないので終わり
	}
	uid := strconv.Itoa(cdata.ID)
	fuid := strconv.Itoa(cdata.FID)

	nowsession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("ap-northeast-1"), //DBのあるリージョン
	}))

	db := dynamodb.New(nowsession, aws.NewConfig().WithRegion("ap-northeast-1"))
	//フレンドリストを持ってくる
	param := &dynamodb.GetItemInput{
		TableName: aws.String("Userdata"),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				N: aws.String(uid),
			},
		},
		AttributesToGet: []*string{
			aws.String("friendlist"),
		},
		ConsistentRead:         aws.Bool(true),
		ReturnConsumedCapacity: aws.String("NONE"),
	}
	data, geterr := db.GetItem(param)
	if geterr != nil {
		fmt.Println(geterr.Error())
		return
	}

	//削除処理
	var list []*dynamodb.AttributeValue
	for i := 0; i < len(data.Item["friendlist"].L); i++ {
		if *data.Item["friendlist"].L[i].N == fuid {
			list = append(data.Item["friendlist"].L[:i], data.Item["friendlist"].L[i+1:]...)
			break
		}
	}

	//更新処理
	param2 := &dynamodb.UpdateItemInput{
		TableName: aws.String("Userdata"),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				N: aws.String(uid), // 既存キー名を指定
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#friendlist": aws.String("friendlist"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":friendlist": {
				L: list,
			},
		},
		ReturnConsumedCapacity: aws.String("NONE"),
		UpdateExpression:       aws.String("set #friendlist = :friendlist"), ////変更の定義
	}
	_, posterr := db.UpdateItem(param2) //登録
	if posterr != nil {
		fmt.Println(posterr.Error())
	}

}

// 友達戦績確認(タスク確認)
func CheckFriend(w rest.ResponseWriter, r *rest.Request) {
	//まずユーザID
	cdata := Friend{}
	recerr := r.DecodeJsonPayload(&cdata)
	if recerr != nil {
		rest.Error(w, recerr.Error(), http.StatusInternalServerError)
		return //受信してないので終わり
	}
	uid := strconv.Itoa(cdata.ID)

	nowsession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("ap-northeast-1"), //DBのあるリージョン
	}))

	db := dynamodb.New(nowsession, aws.NewConfig().WithRegion("ap-northeast-1"))

	//フレンドリストを持ってくる
	param := &dynamodb.GetItemInput{
		TableName: aws.String("Userdata"),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				N: aws.String(uid),
			},
		},
		AttributesToGet: []*string{
			aws.String("friendlist"),
		},
		ConsistentRead:         aws.Bool(true),
		ReturnConsumedCapacity: aws.String("NONE"),
	}
	data, geterr := db.GetItem(param)
	if geterr != nil {
		fmt.Println(geterr.Error())
		return
	}

	//まず友人の人数を送る
	if data.Item["friendlist"] == nil {
		w.WriteJson(0)
	} else {
		w.WriteJson(len(data.Item["friendlist"].L))
		today := time.Now().Year()*10000 + int(time.Now().Month())*100 + time.Now().Day()
		for i := 0; i < len(data.Item["friendlist"].L); i++ {
			//各友人におけるユーザデータ、「完了タスク数/完了+未完(期限前)タスク数」
			var senddata *Friendscore
			param2 := &dynamodb.GetItemInput{
				TableName: aws.String("Userdata"),
				Key: map[string]*dynamodb.AttributeValue{
					"ID": {
						N: aws.String(*data.Item["friendlist"].L[i].N),
					},
				},
				AttributesToGet: []*string{
					aws.String("ID"),
					aws.String("Name"),
					aws.String("Lv"),
					aws.String("exp"),
					aws.String("tasklist"),
				},
				ConsistentRead:         aws.Bool(true),
				ReturnConsumedCapacity: aws.String("NONE"),
			}
			data2, geterr := db.GetItem(param2)
			if geterr != nil {
				fmt.Println(geterr.Error())
				return
			} else {
				senddata = &Friendscore{}
				senddata.ID, _ = strconv.Atoi(*data2.Item["ID"].N)
				senddata.Name = *data2.Item["Name"].S
				senddata.Lv, _ = strconv.Atoi(*data2.Item["Lv"].N)
				senddata.Exp, _ = strconv.Atoi(*data2.Item["exp"].N)
				complete := 0
				all := 0
				if data2.Item["tasklist"] != nil {
					//この友人にタスクがある場合
					for j := 0; j < len(data2.Item["tasklist"].L); j++ {
						//期限を数値化
						deadlines := strings.Split(*data2.Item["tasklist"].L[j].L[3].S, "/")
						year, _ := strconv.Atoi(deadlines[0])
						month, _ := strconv.Atoi(deadlines[1])
						day, _ := strconv.Atoi(deadlines[2])
						deadlineday := year*10000 + month*100 + day
						if deadlineday >= today {
							//期限が切れていないか？
							all++
							if *data2.Item["tasklist"].L[j].L[1].S != "none" {
								//完了しているか
								complete++
							}
						}
					}
				}
				senddata.Complete = complete
				senddata.Total = all
				//送信
				w.WriteJson(senddata)
			}

		}

	}
}
