/**
 * @Author: Nan
 * @Date: 2024/10/11 11:59
 */

package initialize

import (
	"backend/model"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	asyncLogger      *asyncLog
	errorAsyncLogger *asyncLog
)

type asyncLog struct {
	writeChannel     chan []byte
	lumberJackLogger *lumberjack.Logger
	currentDate      string
	mu               sync.Mutex
	stateMu          sync.RWMutex
	wg               sync.WaitGroup
	closeOnce        sync.Once
	closed           bool
	basePath         string
	lastCleanupDate  string
	logPrefix        string
}

func newAsyncLog(basePath, prefix string) *asyncLog {
	al := &asyncLog{
		writeChannel:    make(chan []byte, 500),
		basePath:        basePath,
		logPrefix:       prefix,
		lastCleanupDate: "0000-00-00", // 确保第一次清理生效
	}
	al.updateLogFile()
	al.wg.Add(1)
	zap.L().Info("Starting log processing goroutine")
	go al.processLogEntries()
	return al
}

// 更新日志文件并执行清理
func (al *asyncLog) updateLogFile() {
	al.mu.Lock()
	defer al.mu.Unlock()

	newDate := time.Now().In(model.AppLocation()).Format(time.DateOnly)
	if newDate == al.currentDate {
		return
	}
	// 关闭旧文件
	if al.lumberJackLogger != nil {
		_ = al.lumberJackLogger.Close()
	}
	// 创建新日志文件
	al.currentDate = newDate
	al.lumberJackLogger = &lumberjack.Logger{
		Filename:   filepath.Join(al.basePath, al.logPrefix+"-"+al.currentDate+".log"),
		MaxSize:    20, // MB
		MaxAge:     90, // days
		MaxBackups: 0,  // 不限制数量
		Compress:   false,
		LocalTime:  true,
	}

	// 仅在日期变化时执行清理
	go al.cleanupOldLogs()
}

// 清理过期日志文件
func (al *asyncLog) cleanupOldLogs() {
	al.mu.Lock()
	defer al.mu.Unlock()

	now := time.Now().In(model.AppLocation())
	today := now.Format(time.DateOnly)
	if al.lastCleanupDate == today {
		return
	}

	cutoff := now.AddDate(0, 0, -90)
	pattern := filepath.Join(al.basePath, al.logPrefix+"-*.log*")

	files, _ := filepath.Glob(pattern)
	for _, f := range files {
		base := filepath.Base(f)
		split := strings.SplitN(base, "-", 2)
		if len(split) != 2 {
			continue
		}
		datePart := strings.SplitN(split[1], ".", 2)[0]
		if fileDate, err := time.Parse(time.DateOnly, datePart); err == nil {
			if fileDate.Before(cutoff) {
				_ = os.Remove(f)
				zap.L().Debug("Deleted old log", zap.String("file", f))
			}
		}
	}

	al.lastCleanupDate = today
}

// processLogEntries 处理日志条目
func (al *asyncLog) processLogEntries() {
	defer al.wg.Done()
	for logEntry := range al.writeChannel {
		al.writeLog(logEntry)
	}
}

// 写入日志（确保每日只更新一次日志文件）
func (al *asyncLog) writeLog(logEntry []byte) {
	// updateLogFile保持幂等，并在文件锁内完成日期检查与切换。
	al.updateLogFile()
	al.mu.Lock()
	defer al.mu.Unlock()
	if _, err := al.lumberJackLogger.Write(logEntry); err != nil {
		zap.L().Error("Log write failed", zap.Error(err))
	}
}

// Write 方法，异步写入日志
func (al *asyncLog) Write(p []byte) (n int, err error) {
	logEntry := append([]byte{}, p...) // 确保传入的 p 不会被修改
	al.stateMu.RLock()
	defer al.stateMu.RUnlock()
	if al.closed {
		return 0, os.ErrClosed
	}

	select {
	case al.writeChannel <- logEntry:
		// 正常写入
	default:
		// 防止通道阻塞时丢日志，直接写入文件
		al.writeLog(logEntry)
	}
	return len(p), nil
}

// Close 关闭通道并等待所有日志条目处理完毕
func (al *asyncLog) Close() {
	al.closeOnce.Do(func() {
		al.stateMu.Lock()
		al.closed = true
		close(al.writeChannel)
		al.stateMu.Unlock()

		al.wg.Wait()
		al.mu.Lock()
		if al.lumberJackLogger != nil {
			_ = al.lumberJackLogger.Close()
		}
		al.mu.Unlock()
	})
}

// createLogDirectory 创建日志文件夹
func createLogDirectory(logDir string) {
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		_ = os.MkdirAll(logDir, os.ModePerm)
	}
}

// InitLogger 初始化日志
func InitLogger() *zap.Logger {
	logDir := "./logs"
	createLogDirectory(logDir)
	asyncLogger = newAsyncLog(logDir, "info")
	errorAsyncLogger = newAsyncLog(logDir, "error")
	// 配置日志编码器
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.DateTime) // 自定义时间格式
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.MessageKey = "message"

	// 控制台输出带颜色
	consoleEncoderConfig := encoderConfig
	consoleEncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderConfig)

	// 文件输出不带颜色
	fileEncoderConfig := encoderConfig
	fileEncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	fileEncoder := zapcore.NewJSONEncoder(fileEncoderConfig)

	errorEncoderConfig := encoderConfig
	errorEncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	errorEncoderConfig.StacktraceKey = "stacktrace"
	errorEncoder := zapcore.NewJSONEncoder(errorEncoderConfig)

	// 创建日志核心
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zap.DebugLevel), // 控制台输出
		zapcore.NewCore(fileEncoder, zapcore.AddSync(asyncLogger),
			zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
				return lvl >= zapcore.InfoLevel && lvl < zapcore.ErrorLevel
			})),
		zapcore.NewCore(errorEncoder, zapcore.AddSync(errorAsyncLogger), zap.ErrorLevel),
	)

	// 创建 Logger 对象
	logger := zap.New(core, zap.AddCaller())
	zap.ReplaceGlobals(logger)
	return logger
}

// CloseLogger 关闭日志
func CloseLogger() {
	if asyncLogger != nil {
		asyncLogger.Close()
	}
	if errorAsyncLogger != nil {
		errorAsyncLogger.Close()
	}
}
