package main

import (
	"fmt"
	"github.com/robfig/cron/v3"
	"time"
)

type MyJob struct {
	Name  string
	Count int
}

func (job *MyJob) Run() {
	fmt.Printf("Job %v runing... (%d)\n", job.Name, job.Count)
	job.Count++
}

func main() {
	fmt.Println("practice cron server")

	loc, _ := time.LoadLocation("Asia/Seoul")
	var c = cron.New(
		cron.WithSeconds(),     // 초단위 설정시 설정 필요
		cron.WithLocation(loc), // 서버 시간 설정
		cron.WithChain(
			cron.SkipIfStillRunning(cron.DefaultLogger), // 중복실행시 skip
			//cron.DelayIfStillRunning(cron.DefaultLogger),	// 중복실행시 wait
			cron.Recover(cron.DefaultLogger), //	자동 recover
		),
	)

	myJob := &MyJob{Name: "John"}
	id, err := c.AddJob("*/10 * * * * *", myJob)
	if err != nil {
		fmt.Println("❌ Job 등록 실패:", err)
		return
	}

	fmt.Println("🚀 배치 서버 시작됨")

	// 스케줄 시작
	c.Start()
	// 고루틴으로 2분후 종료처리
	go func() {
		time.Sleep(2 * time.Minute)
		c.Remove(id)
		fmt.Println("job제거 완료")
	}()

	// 종료 방지
	select {}
}
