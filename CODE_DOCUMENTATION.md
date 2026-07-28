# Attendance Collector — সম্পূর্ণ কোড ডকুমেন্টেশন (বাংলায়)

এই ডকুমেন্টেশনটি বিশেষভাবে এমন ডেভেলপারদের জন্য তৈরি করা হয়েছে যারা **Go (Golang)** ভাষায় নতুন। এখানে প্রজেক্টের প্রতিটি ফাইল, ফোল্ডার, কোডের স্ট্রাকচার এবং Go ভাষার বিভিন্ন গুরুত্বপূর্ণ বিষয় সহজ ভাষায় বিস্তারিত ব্যাখ্যা করা হয়েছে।

---

## সূচিপত্র (Table of Contents)
1. [প্রজেক্টের ওভারভিউ (Project Overview)](#১-প্রজেক্টের-ওভারভিউ-project-overview)
2. [Go ভাষার প্রয়োজনীয় মৌলিক বিষয়সমূহ (Go Concepts for Beginners)](#২-go-ভাষার-প্রয়োজনীয়-মৌলিক-বিষয়সমূহ-go-concepts-for-beginners)
3. [ফাইল ও ফোল্ডার স্ট্রাকচার (Directory Structure)](#৩-ফাইল-ও-ফোল্ডার-স্ট্রাকচার-directory-structure)
4. [ফাইলভিত্তিক কোড ব্যাখ্যা (File-by-File Detailed Explanation)](#৪-ফাইলভিত্তিক-কোড-ব্যাখ্যা-file-by-file-detailed-explanation)
   - [main.go](file:///Users/test/Documents/practice/attendance-collector/main.go)
   - [internal/config/config.go](file:///Users/test/Documents/practice/attendance-collector/internal/config/config.go)
   - [internal/models/models.go](file:///Users/test/Documents/practice/attendance-collector/internal/models/models.go)
   - [internal/db/db.go](file:///Users/test/Documents/practice/attendance-collector/internal/db/db.go)
   - [internal/collector/collector.go](file:///Users/test/Documents/practice/attendance-collector/internal/collector/collector.go)
   - [internal/zk/client.go](file:///Users/test/Documents/practice/attendance-collector/internal/zk/client.go)
   - [internal/zk/proto.go](file:///Users/test/Documents/practice/attendance-collector/internal/zk/proto.go)
5. [কমান্ডলাইন ব্যবহার নির্দেশিকা (CLI Commands)](#৫-কমান্ডলাইন-ব্যবহার-নির্দেশিকা-cli-commands)
6. [নতুন Go ডেভেলপারদের জন্য টিপস (Tips for Beginners)](#৬-নতুন-go-ডেভেলপারদের-জন্য-টিপস-tips-for-beginners)

---

## ১. প্রজেক্টের ওভারভিউ (Project Overview)

`attendance-collector` হলো একটি হালকা কিন্তু শক্তিশালী গো (Go) সার্ভিস বা CLI অ্যাপ্লিকেশন। 
এর মূল কাজ হলো:
1. **ZKTeco বায়োমেট্রিক ডিভাইস** (আঙুলের ছাপ বা ফেস রিকগনিশন মেশিন) থেকে কর্মচারীদের তালিকা (Employees) এবং উপস্থিতির সময় (Attendance Punches) TCP নেটওয়ার্কের মাধ্যমে সংগ্রহ করা।
2. সংগৃহীত তথ্য **SQLite** অথবা **MySQL** ডাটাবেসে সেভ করে রাখা।
3. **Deduplication (ডুপ্লিকেট প্রতিরোধ)**: একই উপস্থিতির রেকর্ড যেন ডাটাবেসে একাধিকবার সেভ না হয়, সেটির জন্য SHA-256 হ্যাশ ব্যবহার করা।
4. **Daemon / Background Runner**: ব্যাকগ্রাউন্ডে নিয়মিত নির্দিষ্ট সময় পর পর একাধিক ডিভাইস থেকে ডেটা সংগ্রহ করা (Staggered Schedule)।

---

## ২. Go ভাষার প্রয়োজনীয় মৌলিক বিষয়সমূহ (Go Concepts for Beginners)

আপনি যদি Go-তে নতুন হন, তবে এই কোডটিতে নিচের কনসেপ্টগুলো দেখতে পাবেন:

### ক. Package (প্যাকেজ)
Go-তে প্রতিটি `.go` ফাইলের শুরুতে একটি `package` ডিক্লেয়ার করতে হয়। 
- `main` প্যাকেজ নির্দেশ করে এটি একটি এক্সিকিউটেবল (সংকলনযোগ্য) প্রোগ্রাম যার মধ্যে `main()` ফাংশন আছে।
- `internal/` ফোল্ডারের অধীনে থাকা প্যাকেজগুলো (`config`, `models`, `db`, `collector`, `zk`) শুধুমাত্র এই প্রজেক্টের ভেতরেই ব্যবহার করা যাবে, বাইরের অন্য কোনো Go প্রজেক্ট সরাসরি এগুলো ইম্পোর্ট করতে পারবে না।

### খ. Struct (স্ট্রাক্ট) এবং Method Receiver
Go-তে অন্যান্য অবজেক্ট-ওরিয়েন্টেড ভাষার মতো `class` নেই। তার বদলে `struct` ব্যবহার করে ডাটা ফিল্ড তৈরি করা হয়।
```go
type DeviceConfig struct {
    Name string
    IP   string
    Port uint
}
```
একটি Struct-এর সাথে ফাংশন যুক্ত করতে **Receiver** ব্যবহার করা হয়:
```go
func (d DeviceConfig) Address() string {
    return fmt.Sprintf("%s:%d", d.IP, d.Port)
}
```

### গ. Pointer (পয়েন্টার - `*`)
Go-তে `*Config` বা `*gorm.DB` এর অর্থ হলো কোনো ভ্যালুর মেমোরি এড্রেস রেফারেন্স করা। 
- পয়েন্টার ব্যবহার করলে ডাটা মেমোরিতে কপি না হয়ে সরাসরি মূল ভ্যালুতে কাজ করে, যা মেমোরি এবং পারফরম্যান্স বাঁচায়।

### ঘ. Error Handling (এরর হ্যান্ডলিং)
Go-তে Exception handling (`try-catch`) নেই। ফাংশন দুটি ভ্যালু রিটার্ন করে: রেসাল্ট এবং এরর (`err`)।
```go
cfg, err := config.Load()
if err != nil {
    // এরর হলে হ্যান্ডেল করা
}
```

### ঙ. Goroutine ও Channel (`go func()`, `select`, `time.Ticker`)
- **Goroutine (`go myFunc()`)**: ব্যাকগ্রাউন্ডে থ্রেড হিসেবে ফাংশন চালানোর জন্য।
- **`select` & `time.Ticker`**: সময়ে সময়ে নির্দিষ্ট কাজ করার জন্য এবং সংকেত পাওয়ার জন্য ব্যবহৃত হয়।
- **`sync.WaitGroup`**: ব্যাকগ্রাউন্ডের সব কাজ শেষ হওয়া পর্যন্ত মেইন প্রোগ্রামকে অপেক্ষা করানোর জন্য।

---

## ৩. ফাইল ও ফোল্ডার স্ট্রাকচার (Directory Structure)

```
attendance-collector/
├── main.go                     # CLI এন্ট্রি পয়েন্ট এবং ডেমন চালক
├── Dockerfile                  # প্রোডাকশন-গ্রেড মাল্টি-স্টেজ ডকার বিল্ড ফাইল
├── .dockerignore               # ডকার কনটেক্সট ফিল্টারিং ফাইল
├── RUN_GUIDE.md                # Docker এবং Native ডেপ্লয়মেন্ট গাইড
├── CODE_DOCUMENTATION.md       # পূর্ণাঙ্গ কোড ডকুমেন্টেশন (বাংলায়)
├── go.mod                      # ডিপেনডেন্সি মডিউল ফাইল
├── go.sum                      # ডিপেনডেন্সি লক ফাইল
├── .env.example                # এনভায়রনমেন্ট কনফিগারেশন টেমপ্লেট
└── internal/                   # ইন্টারনাল প্যাকেজসমূহ
    ├── config/
    │   └── config.go           # .env এবং পরিবেশগত ভ্যারিয়েবল লোডার
    ├── models/
    │   └── models.go           # ডাটাবেস স্কিমা / টেবিল মডেল
    ├── db/
    │   └── db.go               # GORM কানেকশন ও অটো-মাইগ্রেশন
    ├── collector/
    │   └── collector.go        # ডিভাইস ও ডাটাবেসের মধ্যে সমন্বয়কারী (Business Logic)
    └── zk/
        ├── client.go           # ZKTeco ক্লায়েন্টের High-level API (Users, Attendance, Time)
        └── proto.go            # ZKTeco Low-level TCP বাইনারি প্রোটোকল ও প্যাকেট ফিল্টারিং
```

---

## ৪. ফাইলভিত্তিক কোড ব্যাখ্যা (File-by-File Detailed Explanation)

### ১. [main.go](file:///Users/test/Documents/practice/attendance-collector/main.go)
এটি প্রোগ্রামের মূল শুরু বা এন্ট্রি পয়েন্ট।

#### প্রধান অংশগুলো:
1. **`main()` ফাংশন (লাইন ২৬-৬২)**:
   - কমান্ডলাইন আর্গুমেন্ট পর্যবেক্ষণ করে (যেমন: `./attendance-collector pull` বা `run`)।
   - [config.Load()](file:///Users/test/Documents/practice/attendance-collector/internal/config/config.go#L54) ডেকে পরিবেশ থেকে সেটআপ রিড করে।
   - [db.Open(cfg)](file:///Users/test/Documents/practice/attendance-collector/internal/db/db.go#L20) দিয়ে ডাটাবেস কানেকশন খোলে।
   - [collector.New(...)](file:///Users/test/Documents/practice/attendance-collector/internal/collector/collector.go#L24) দিয়ে কানেক্টর তৈরি করে এবং `switch cmd` দিয়ে নির্ধারিত কমান্ড চালায়।

2. **`runDaemon(...)` ফাংশন (প্রোডাকশন গ্রেড অটো শিডিউলিং)**:
   - ব্যাকগ্রাউন্ড ডেমন মোড। প্রতিটি ZKTeco ডিভাইসের জন্য আলাদা **Goroutine** তৈরি করা হয়।
   - **১০ মিনিট পর পর পুল (Poll Interval)**: প্রতি ১০ মিনিট পর পর ডিভাইস থেকে উপস্থিতির ডাটা রিড করা হয় (`POLL_INTERVAL_MINUTES=10`)।
   - **৫ মিনিটের স্ট্যাগার (Stagger Offset)**: ডিভাইস ১ (ডিভাইস ইন / Outside) শুরুতেই (T+0m) ডাটা টানা শুরু করে। ৫ মিনিট পর (T+5m) ডিভাইস ২ (ডিভাইস আউট / Inside) ডাটা টানা শুরু করে (`STAGGER_MINUTES=5`)। এর ফলে নেটওয়ার্কে কখনও দুটি ডিভাইস একসাথে ওভারল্যাপ করে কাজ করে না।
   - **Panic Recovery (প্যানিক সেফটি)**: কোনো নেটওয়ার্ক ত্রুটি বা আনপ্রসেসড এররের কারণে Goroutine যেন ক্র্যাশ না করে, সে জন্য `recover()` দিয়ে প্যানিক সেফগার্ড যুক্ত করা হয়েছে।
   - **Non-Overlapping Execution (TryLock)**: স্লো নেটওয়ার্কের কারণে ১০ মিনিটেও একটি পুল শেষ না হলে পরবর্তী টিংকার সাথে সাথে ওভারল্যাপ না করে স্কিপ করে লগে জানিয়ে দেয়।
   - `signal.NotifyContext` ব্যবহার করে **Graceful Shutdown** নিশ্চিত করা হয়েছে।
   - প্রতি ৬ ঘণ্টা পর পর স্বয়ংক্রিয়ভাবে Employee Sync চালিত হয়।

---

### ২. [internal/config/config.go](file:///Users/test/Documents/practice/attendance-collector/internal/config/config.go)
অ্যাপ্লিকেশনের সমস্ত কনফিগারেশন এবং `.env` ফাইল রিড করার কাজ এই ফাইলটি করে।

#### প্রধান অংশগুলো:
1. **`DeviceConfig` Struct (লাইন ১৭-২২)**:
   - একটি ডিভাইসের নাম, IP, Port এবং দিক (`in`/`out`) ধরে রাখে।
2. **`Config` Struct (লাইন ৩০-৫১)**:
   - ডাটাবেসের ড্রাইভার (`sqlite`/`mysql`), কানেকশন ডিটেইলস, ডিভাইসের তালিকা, টাইমআউট, রিট্রাই কাউন্ট ইত্যাদি ধরে রাখে।
3. **`Load()` ফাংশন (লাইন ৫৪-১০০)**:
   - `godotenv.Load()` ব্যবহার করে `.env` ফাইল পড়ে।
   - ডিফোল্ট ভ্যালু সেট করে (যেমন `POLL_INTERVAL_MINUTES` না থাকলে ১০ মিনিট)।
   - `ZK_IN_IP` ও `ZK_OUT_IP` কনফিগার থাকলে ডিভাইসগুলোকে `Devices` স্লাইসে যোগ করে।

---

### ৩. [internal/models/models.go](file:///Users/test/Documents/practice/attendance-collector/internal/models/models.go)
ডাটাবেস টেবিল এবং স্কিমা ডিফাইন করা হয়েছে এখানে।

#### টেবিল সমুহ:
1. **`Device` Struct (লাইন ১৪-২৮)**:
   - `devices` টেবিল। ZKTeco ডিভাইসের IP, স্ট্যাটাস, শেষ ডাটা পুল করার সময় ইত্যাদি তথ্য সংরক্ষণ করে।
2. **`Employee` Struct (লাইন ৩৩-৪৩)**:
   - `employees` টেবিল। ডিভাইসে রেজিস্টার্ড কর্মচারীদের PIN (`device_user_pin`), নাম, কার্ড নম্বর ইত্যাদি রাখে।
3. **`Attendance` Struct (লাইন ৪৮-৫৮)**:
   - `attendances` টেবিল। প্রতিবার পাঞ্চ বা উপস্থিতির রেকর্ড রাখে।

#### Deduplication Hash (ডুপ্লিকেট রো প্রতিরোধ):
- **`MakeDedupeHash()` ফাংশন (লাইন ৬৭-৭৫)**:
  - `sha256( "<deviceID>|<pin>|<YYYY-MM-DD HH:MM:SS>" )` ফরম্যাটে একটি ইউনিক হ্যাশ তৈরি করে।
  - এটি ডাটাবেসে `dedupe_hash` হিসেবে সেভ হয়। এর ফলে একই পাঞ্চ একাধিকবার পড়লেও ডাটাবেসে ডুপ্লিকেট এন্ট্রি হয় না।

---

### ৪. [internal/db/db.go](file:///Users/test/Documents/practice/attendance-collector/internal/db/db.go)
GORM ORM ব্যবহার করে ডাটাবেসের সাথে যুক্ত হওয়ার ফাইল।

#### প্রধান অংশগুলো:
1. **`Open(c *config.Config)` (লাইন ২০-৫৫)**:
   - `c.DBDriver` চেক করে SQLite বা MySQL ডায়ালেক্টর নির্বাচন করে।
   - `c.AutoMigrate` যদি `true` থাকে, তবে স্বয়ংক্রিয়ভাবে `devices`, `employees`, এবং `attendances` টেবিলগুলো তৈরি বা আপডেট করে।

---

### ৫. [internal/collector/collector.go](file:///Users/test/Documents/practice/attendance-collector/internal/collector/collector.go)
প্রজেক্টের মূল বিজনেস লজিক হাব। এটি `zk` প্যাকেজের মাধ্যমে ডিভাইস থেকে ডাটা আনে এবং `models` দিয়ে ডাটাবেসে সেভ করে।

#### প্রধান অংশগুলো:
1. **`SyncEmployees()` (লাইন ৭৩-১২১)**:
   - ডিভাইসের সব ইউজারদের নিয়ে আসে।
   - GORM-এর `clause.OnConflict` ব্যবহার করে ইউজার থাকলে নাম/কার্ড আপডেট করে, না থাকলে নতুন এন্ট্রি তৈরি করে।
2. **`PullAttendance()` (লাইন ১২৪-১৪৫)**:
   - ডিভাইস থেকে উপস্থিতির সব রেকর্ড সংগ্রহ করে।
   - `storePunches()` ফাংশনে পাঠিয়ে ফিল্টারিং ও সেভ করার কাজ সম্পন্ন করে।
3. **`storePunches()` (লাইন ১৪৯-২২১)**:
   - **ব্যাচ প্রসেসিং (Chunking)**: একবারে ৫০০টি করে ডাটা ফিল্টার করে চেক করা হয় ডাটাবেসে হ্যাশটি আগেই আছে কি না।
   - শুধু নতুন পাঞ্চগুলো ডাটাবেসে একসাথে ইনসার্ট করা হয়।
4. **`ClearAttendance()` (লাইন ২৭০-২৭৯)**:
   - ডিভাইসের মেমোরি খালি করার জন্য কাজ করে (অবশ্যই ডাটাবেসে সেভ করার পর)।

---

### ৬. [internal/zk/client.go](file:///Users/test/Documents/practice/attendance-collector/internal/zk/client.go)
ZKTeco ডিভাইস ক্লায়েন্টের একটি ইউজার-ফ্রেন্ডলি প্যাকেজ।

#### প্রধান অংশগুলো:
1. **`withConnection()` (লাইন ৪৯-৭৩)**:
   - ডিভাইসের সাথে কানেকশন তৈরি করে কাজ সম্পন্ন করে এবং কাজ শেষে অবশ্যই কানেকশন ক্লোজ করে।
   - কানেকশন ব্যর্থ হলে স্বয়ংক্রিয়ভাবে রিট্রাই (Retry) করে।
2. **`Time()`**, **`Users()`**, **`Attendance()`**, **`ClearAttendance()`**:
   - ডিভাইস থেকে যথাক্রমে সময়, ব্যবহারকারী তালিকা, পাঞ্চ লগ পড়া এবং ক্লিয়ার করার উচ্চতর ফাংশনসমূহ।

---

### ৭. [internal/zk/proto.go](file:///Users/test/Documents/practice/attendance-collector/internal/zk/proto.go)
ZKTeco প্রোপাইটারি (Proprietary) TCP প্যাকেটের লো-লেভেল ইমপ্লিমেন্টেশন। এটি PyZK পাইথন লাইব্রেরির আদলে Go-তে তৈরি করা হয়েছে।

#### প্রধান অংশগুলো:
1. **Protocol Constants (লাইন ১১-৩৫)**: ZKTeco হেডার কমান্ড কোডসমূহ (যেমন `cmdConnect = 1000`, `cmdAckOK = 2000`)।
2. **`dial()` (লাইন ৪৯-৭৬)**: TCP এর মাধ্যমে কানেক্ট করে হ্যান্ডশেক ও পাসওয়ার্ড পাস করে।
3. **`buildHeader()` & `checksum()` (লাইন ৮৮-১০৬, ২৬৩-২৮৫)**: пакеট পাঠানোর আগে ১৬-বিট ওয়ান্স কমপ্লিমেন্ট চেকমার্ক বা চেকসাম হিসেব করে।
4. **`readN()` & `readPacket()`**: TCP স্ট্রিম থেকে নির্দিষ্ট সাইজের সম্পূর্ণ বাইনারি প্যাকেট রিড করা নিশ্চিত করে।
5. **`decodeTime()` (লাইন ৩১২-৩২৬)**: ৪-বাইটের এনকোড করা ZKTeco বাইনারি টাইমকে Go-এর standard `time.Time`-এ রূপান্তর করে।

---

## ৫. কমান্ডলাইন ব্যবহার নির্দেশিকা (CLI Commands)

আপনি টার্মিনালে নিচের কমান্ডগুলো ব্যবহার করে অ্যাপ্লিকেশন চালাতে পারেন:

| কমান্ড (Command) | কাজ (Description) |
|---|---|
| `./attendance-collector test` | ডিভাইসগুলোর সংযোগ এবং সময় ঠিক আছে কিনা তা পরীক্ষা করে। |
| `./attendance-collector pull` | সমস্ত ডিভাইস থেকে উপস্থিতি টেনে এনে ডাটাবেসে সেভ করে। |
| `./attendance-collector sync-employees` | ডিভাইস থেকে কর্মচারীদের তালিকা এনে ডাটাবেসে যোগ বা আপডেট করে। |
| `./attendance-collector run` | ব্যাকগ্রাউন্ডে ডেমন হিসেবে চালু হয় এবং নিয়মিত ডাটা সংগ্রহ করে। |
| `./attendance-collector clear-device` | ডিভাইস থেকে সব পাঞ্চ লগ মুছে ফেলে (ডাটাবেসে সেভ হওয়ার পর)। |

**ফিল্টার অপশন**: `--device=in` অথবা `--device=192.168.1.201` যোগ করে নির্দিষ্ট কোনো ডিভাইসে কমান্ড চালাতে পারেন।

---

## ৬. নতুন Go ডেভেলপারদের জন্য টিপস (Tips for Beginners)

১. **Error handling কখনো ইগনোর করবেন না**:
   Go-তে `_ = err` না করে সর্বদা `if err != nil` চেক করা একটি উত্তম চর্চা।
২. **Slice Slicing বোঝা**:
   `data[8:]` বা `r[0:2]` দিয়ে বাইট এরে বা স্লাইস থেকে নির্দিষ্ট অংশ কাটা হয়। 
৩. **Binary Data & Endianness**:
   `binary.LittleEndian.Uint16(buf)` দিয়ে বাইট এরে থেকে নম্বর রিড করা হয়। ZKTeco রিভার্স/লিটল-এন্ডিয়ান ফরম্যাটে ডাটা পাঠায়।
৪. **Concurrency Safety**:
   Goroutine ব্যবহার করার সময় রেস কন্ডিশন এড়াতে `sync.WaitGroup` এবং `context.Context` ব্যবহার নিরাপদ।

---
*ডকুমেন্টেশন তৈরীর তারিখ: ২০ জুলাই, ২০২৬*
