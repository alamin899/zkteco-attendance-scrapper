# Attendance Collector — চালান এবং ডেপ্লয়মেন্ট গাইড (Run & Deployment Guide)

এই গাইডে **Docker ব্যবহার করে** এবং **Docker ছাড়া সরাসরি (Native)** অ্যাপ্লিকেশনটি কিভাবে চালনা করবেন তা সহজ ভাষায় ধাপে ধাপে বর্ণনা করা হলো।

---

## 📋 সূচিপত্র (Table of Contents)
1. [পদ্ধতি ১: Docker দিয়ে চালানো (Recommended for Production)](#১-docker-দিয়ে-চালানো-recommended-for-production)
   - [১.১ ডকার ইমেজ তৈরি (Docker Build)](#১১-ডকার-ইমেজ-তৈরি-docker-build)
   - [১.২ ডেমন মোডে ব্যাকগ্রাউন্ডে চালানো (Run Daemon Container)](#১২-ডেমন-মোডে-ব্যাকগ্রাউন্ডে-চালানো-run-daemon-container)
   - [১.৩ একক কমান্ড চালানো (Run One-shot CLI Commands)](#১৩-একক-কমান্ড-চালানো-run-one-shot-cli-commands)
   - [১.৪ কন্টেইনার কন্ট্রোল ও লগ দেখা (Container Control & Logs)](#১৪-কন্টেইনার-কন্ট্রোল-ও-লগ-দেখা-container-control--logs)
2. [পদ্ধতি ২: Docker ছাড়া সরাসরি চালানো (Native Run)](#২-docker-ছাড়া-সরাসরি-চালানো-native-run)
   - [২.১ সোর্স কোড থেকে প্রজেক্ট বিল্ড (Build Binary)](#২১-সোর্স-কোড-থেকে-প্রজেক্ট-বিল্ড-build-binary)
   - [২.২ সরাসরি CLI কমান্ড চালানো (Run CLI Commands)](#২২-সরাসরি-cli-কমান্ড-চালানো-run-cli-commands)
   - [২.৩ Systemd সার্ভিস হিসেবে প্রসেস চালানো (Linux Production Service)](#২৩-systemd-সার্ভিস-হিসেবে-প্রসেস-চালানো-linux-production-service)
3. [কনফিগারেশন নোটস (Network & Environment Notes)](#৩-কনফিগারেশন-নোটস-network--environment-notes)

---

## ১. Docker দিয়ে চালানো (Recommended for Production)

প্রজেক্টটিতে **Production-Grade Multi-stage Dockerfile** ইমপ্লিমেন্ট করা আছে, যা অত্যন্ত হালকা (Lightweight) এবং নিরাপদ (Non-root user)।

### ১.১ ডকার ইমেজ তৈরি (Docker Build)

টার্মিনালে প্রজেক্ট ডিরেক্টরিতে গিয়ে নিচের কমান্ডটি রান করে ডকার ইমেজ বিল্ড করুন:

```bash
docker build -t attendance-collector .
```

---

### ১.২ ডেমন মোডে ব্যাকগ্রাউন্ডে চালানো (Run Daemon Container)

সার্ভারে ১০ মিনিট পর পর অটো-শিডিউল অনুযায়ী ব্যাকগ্রাউন্ডে চালানোর জন্য নিচের কমান্ডটি দিন:

#### ক) MySQL ডাটাবেস ব্যবহার করলে (Local MySQL on port 35060):
> **নোট:** `--net=host` ফ্ল্যাগ ব্যবহার করলে কন্টেইনারটি হোস্ট মেশিনের `127.0.0.1:35060` এবং ZKTeco ডিভাইস IP-তে সরাসরি কানেক্ট করতে পারবে।

```bash
docker run -d \
  --name attendance-daemon \
  --restart always \
  --net=host \
  --env-file .env \
  attendance-collector run
```

#### খ) SQLite ডাটাবেস ব্যবহার করলে (Volume Mapping):
SQLite ফাইলটি কন্টেইনার রিস্টার্ট হলেও যেন ডাটা সেভ থাকে, সে জন্য Volume মাউন্ট করতে হবে:

```bash
docker run -d \
  --name attendance-daemon \
  --restart always \
  --net=host \
  --env-file .env \
  -v $(pwd)/attendance.db:/app/attendance.db \
  attendance-collector run
```

---

### ১.৩ একক কমান্ড চালানো (Run One-shot CLI Commands)

আপনার প্রয়োজনে কন্টেইনার তৈরি করে নির্দিষ্ট কাজ সম্পন্ন করার পর স্বয়ংক্রিয়ভাবে মুছে ফেলার জন্য (`--rm`) নিচের কমান্ডগুলো ব্যবহার করতে পারেন:

1. **ডিভাইস সংযোগ পরীক্ষা করা (Test Connection):**
   ```bash
   docker run --rm --net=host --env-file .env attendance-collector test
   ```

2. **একবার উপস্থিতির রেকর্ড টানা (Pull Attendance Once):**
   ```bash
   docker run --rm --net=host --env-file .env attendance-collector pull
   ```

3. **এমপ্লয়ি তথ্য সিঙ্ক করা (Sync Employees):**
   ```bash
   docker run --rm --net=host --env-file .env attendance-collector sync-employees
   ```

4. **নির্দিষ্ট ডিভাইসের মেমোরি খালি করা (Clear Device Log):**
   ```bash
   docker run --rm --net=host --env-file .env attendance-collector clear-device --device=in
   ```

---

### ১.৪ কন্টেইনার কন্ট্রোল ও লগ দেখা (Container Control & Logs)

- **লাইভ লগ বা আউটপুট দেখতে:**
  ```bash
  docker logs -f attendance-daemon
  ```

- **কন্টেইনার রিস্টার্ট করতে:**
  ```bash
  docker restart attendance-daemon
  ```

- **কন্টেইনার বন্ধ বা মুছতে:**
  ```bash
  docker stop attendance-daemon
  docker rm attendance-daemon
  ```

---

## ২. Docker ছাড়া সরাসরি চালানো (Native Run)

কোনো ডকার ছাড়া সরাসরি সার্ভার বা লোকাল মেশিনে চালাতে চাইলে নিচের পদ্ধতি অনুসরণ করুন।

### ২.১ সোর্স কোড থেকে প্রজেক্ট বিল্ড (Build Binary)

প্রয়োজনীয় ডিপেনডেন্সি: **Go 1.24+**

```bash
# প্রজেক্ট ডিরেক্টরিতে যান
cd attendance-collector

# ডিপেনডেন্সি ডাউনলোড ও বিল্ড করুন
go mod download
go build -o attendance-collector main.go
```

বিল্ড সম্পন্ন হলে `attendance-collector` নামের একটি সিঙ্গেল এক্সিকিউটেবল বাইনারি ফাইল তৈরি হবে।

---

### ২.২ সরাসরি CLI কমান্ড চালানো (Run CLI Commands)

1. **কানেক্টিভিটি টেস্ট (Test Connectivity):**
   ```bash
   ./attendance-collector test
   ```

2. **ইউজার/এমপ্লয়ি ডাটা সিঙ্ক (Sync Employees):**
   ```bash
   ./attendance-collector sync-employees
   ```

3. **একবার উপস্থিতির তথ্য টানা (One-time Attendance Pull):**
   ```bash
   ./attendance-collector pull
   ```

4. **নির্দিষ্ট ডিভাইসের তথ্য ব্যাকগ্রাউন্ডে সেভ করে ক্লিয়ার করা:**
   ```bash
   ./attendance-collector clear-device --device=192.168.10.201
   ```

5. **ডেমন মোডে ব্যাকগ্রাউন্ডে চালানো (Auto-Schedule Daemon):**
   ```bash
   ./attendance-collector run
   ```

---

### ২.৩ macOS-এ ল্যাপটপ অন হলে অটো-স্টার্ট চালু করা (macOS LaunchDaemon)

> ⚠️ **গুরুত্বপূর্ণ নোট**: macOS 15 (Sequoia) বা নতুন ম্যাক-এ "Local Network Privacy" এর কারণে `LaunchAgent` (ইউজার লেভেল) ব্যবহার করলে ব্যাকগ্রাউন্ড প্রসেস নেটওয়ার্কে কানেক্ট করতে পারে না ("no route to host" দেখায়)। এর সমাধান হিসেবে একে `LaunchDaemon` (সিস্টেম রুট লেভেল) হিসেবে সেটআপ করতে হবে।

#### ল্যাপটপ অন হলে অটো-স্টার্ট সেটআপ করার নিয়ম:

১. টার্মিনালে নিচের কমান্ডটি দিয়ে LaunchDaemon `.plist` ফাইল তৈরি করুন (পাসওয়ার্ড চাইতে পারে):

```bash
sudo nano /Library/LaunchDaemons/com.attendance.collector.plist
```

২. ফাইলে নিচের XML কোডটি পেস্ট করুন:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.attendance.collector</string>
    
    <key>ProgramArguments</key>
    <array>
        <string>/Users/test/Documents/practice/attendance-zkteco-all/AttendanceCollector_Release/attendance-collector</string>
        <string>run</string>
    </array>
    
    <key>WorkingDirectory</key>
    <string>/Users/test/Documents/practice/attendance-zkteco-all/AttendanceCollector_Release</string>
    
    <key>RunAtLoad</key>
    <true/>
    
    <key>KeepAlive</key>
    <true/>
    
    <key>StandardOutPath</key>
    <string>/Users/test/Documents/practice/attendance-zkteco-all/AttendanceCollector_Release/daemon.log</string>
    
    <key>StandardErrorPath</key>
    <string>/Users/test/Documents/practice/attendance-zkteco-all/AttendanceCollector_Release/daemon.err.log</string>
</dict>
</plist>
```

৩. সার্ভিসটি লোড এবং চালু করুন:

```bash
# সার্ভিস চালু করা (ল্যাপটপ রিবুট করলেও এটি অটো রান করবে)
sudo launchctl load /Library/LaunchDaemons/com.attendance.collector.plist

# সার্ভিস স্ট্যাটাস দেখা (0 স্ট্যাটাস দেখালে সার্ভিস ঠিকমতো চলছে)
sudo launchctl list | grep com.attendance.collector

# ব্যাকগ্রাউন্ড লগ দেখতে (কোড কাজ করছে কিনা)
tail -n 20 /Users/test/Documents/practice/attendance-zkteco-all/AttendanceCollector_Release/daemon.log

# সার্ভিস বন্ধ বা রিমুভ করা
sudo launchctl unload /Library/LaunchDaemons/com.attendance.collector.plist
```

```

---

### ২.৪ বিকল্প উপায়: macOS Automator App + Login Items (No Sudo / No Terminal)

যদি টার্মিনাল বা `sudo` পাসওয়ার্ড ব্যবহার করতে না চান, তবে এটি সবচেয়ে নিরাপদ ও সহজ উপায়। 
আপনাকে ৩টি ফাইল একটি ফোল্ডারে (যেমন: `AttendanceCollector_Release`) করে ক্লায়েন্টকে দিতে হবে:
১. `attendance-collector` (আপনার বাইনারি ফাইল)
২. `.env` (ডাটাবেস ও আইপি কনফিগারেশন)
৩. `AttendanceStarter.app` (আপনার বানানো Automator শর্টকাট অ্যাপ)

**কিভাবে এই পোর্টেবল AttendanceStarter.app বানাবেন?**

১. আপনার Mac-এ **Automator** অ্যাপটি ওপেন করুন।
২. নতুন ডকুমেন্টে **Application** সিলেক্ট করুন।
৩. বাম পাশ থেকে **"Run Shell Script"** সার্চ করে ডান পাশে টেনে আনুন।
৪. সেখানে নিচের কোডটি হুবহু কপি করে পেস্ট করুন (এখানে `$HOME` ব্যবহার করা হয়েছে, যেন যেকোনো ইউজারের ল্যাপটপেই এটি একা একা সঠিক ফোল্ডার চিনে নেয়):
   ```bash
   nohup $HOME/AttendanceCollector_Release/attendance-collector run > $HOME/AttendanceCollector_Release/daemon.log 2>&1 &
   ```
৫. **File > Save** এ গিয়ে ফাইলটির নাম দিন `AttendanceStarter.app` এবং `AttendanceCollector_Release` ফোল্ডারের ভেতরেই সেভ করুন।

**ক্লায়েন্ট কীভাবে সেটআপ করবে?**
ক্লায়েন্টকে বলবেন এই ফোল্ডারটি যেন তার Mac-এর Home ফোল্ডারে (বা ডকুমেন্টস-এ, তবে Home ফোল্ডার বেস্ট) রাখে। এরপর:
১. Mac এর **System Settings > General > Login Items** -এ যাবে।
২. **"Open at Login"** এর নিচে `+` আইকনে ক্লিক করে `AttendanceStarter.app` টি অ্যাড করে দিবে।

ল্যাপটপ অন হলে এটি একবার অটো-ওপেন হয়ে ব্যাকগ্রাউন্ডে কোড রান করে দিবে, এবং অ্যাপল একে কোনো ব্লক করবে না!

---

### ২.৫ Linux সার্ভারে Systemd সার্ভিস হিসেবে প্রসেস চালানো (Linux Production Service)

Linux (Ubuntu/Debian/CentOS) সার্ভারে ব্যাকগ্রাউন্ডে চালানোর জন্য Systemd ফাইল:

১. সার্ভারে সার্ভিস ফাইল তৈরি করুন:
```bash
sudo nano /etc/systemd/system/attendance-collector.service
```

২. নিচের কনফিগারেশনটি বসিয়ে সেভ করুন:
```ini
[Unit]
Description=ZKTeco Attendance Collector Service
After=network.target mysql.service

[Service]
Type=simple
User=root
WorkingDirectory=/path/to/attendance-collector
ExecStart=/path/to/attendance-collector/attendance-collector run
Restart=always
RestartSec=10
EnvironmentFile=/path/to/attendance-collector/.env

[Install]
WantedBy=multi-user.target
```

৩. সার্ভিস এনেবল ও চালু করুন:
```bash
sudo systemctl daemon-reload
sudo systemctl enable attendance-collector
sudo systemctl start attendance-collector

# সার্ভিস স্ট্যাটাস দেখতে
sudo systemctl status attendance-collector

# লাইভ লগ দেখতে
sudo journalctl -u attendance-collector -f
```

---

## ৩. কনফিগারেশন নোটস (Network & Environment Notes)

1. **`127.0.0.1:35060` (Local MySQL Connection)**:
   ডকারে লুপব্যাক আইপি (`127.0.0.1`) ব্যবহারের সময় `--net=host` নেটওয়ার্ক ড্রাইভার ব্যবহার করা জরুরি, যেন কন্টেইনারটি সরাসরি হোস্ট প্রসেসের সাথে কানেক্ট হতে পারে।

2. **ZKTeco Port (4370)**:
    নিশ্চিত করুন আপনার ফায়ারওয়ালে TCP Port 4370 ব্লক করা নেই, যাতে ডিভাইসগুলোর সাথে সংযোগ স্থাপন করতে সমস্যা না হয়।

3. **.env ফাইল এনভায়রনমেন্ট পাসিং**:
   ডিভাইসের আইপি বা ডাটাবেসের তথ্য পরিবর্তন করতে শুধুমাত্র `.env` ফাইল আপডেট করাই যথেষ্ট।
