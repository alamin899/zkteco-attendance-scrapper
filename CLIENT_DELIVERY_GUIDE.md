# Attendance Collector — ক্লায়েন্ট ডেলিভারি ও সেটআপ গাইড (Client Delivery & Closed-Source Setup Guide)

এই ডকুমেন্টেশনে সোর্স কোড (Source Code) গোপন রেখে কিভাবে অ্যাপ্লিকেশনের কমপাইল্ড ভার্সন ক্লায়েন্টকে প্রদান করবেন এবং ক্লায়েন্টের ল্যাপটপে ধাপে ধাপে সেটআপ করবেন তা বিস্তারিত বর্ণনা করা হলো।

---

## 🔒 কেন এই পদ্ধতিতে কোড ১০০% সুরক্ষিত থাকে?

Go (Golang) দিয়ে বিল্ড করা প্রোগ্রামগুলোকে সোর্স কোড ছাড়া সরাসরি একটি **সিঙ্গেল এক্সিকিউটেবল বাইনারি ফাইলে (`.exe` বা বাইনারি)** রূপান্তর করা হয়।
- ক্লায়েন্টকে কোনো `.go` ফাইল দেওয়া হয় না।
- ক্লায়েন্ট আপনার কোডের কোনো লজিক দেখতে পারবে না।
- ক্লায়েন্টের ল্যাপটপে Go ইনস্টল করার কোনো প্রয়োজন নেই।

---

## 📦 ধাপ ১: ক্লায়েন্ট ডেলিভারি প্যাকেজ তৈরি (Developer Build Steps)

আপনার ডেভেলপমেন্ট পিসিতে ক্লায়েন্টের অপারেটিং সিস্টেম অনুযায়ী নিচের যেকোনো একটি কমান্ড দিয়ে প্রজেক্ট বিল্ড করুন:

### ১.১ কমপাইল্ড বাইনারি তৈরি (Cross-Compilation)

- **ক্লায়েন্ট Mac (Apple Silicon M1/M2/M3) হলে:**
  ```bash
  GOOS=darwin GOARCH=arm64 go build -ldflags="-w -s" -o attendance-collector main.go
  ```

- **ক্লায়েন্ট Mac (Intel Core i5/i7) হলে:**
  ```bash
  GOOS=darwin GOARCH=amd64 go build -ldflags="-w -s" -o attendance-collector main.go
  ```

- **ক্লায়েন্ট Windows ব্যবহার করলে:**
  ```bash
  GOOS=windows GOARCH=amd64 go build -ldflags="-w -s" -o attendance-collector.exe main.go
  ```

- **ক্লায়েন্ট Linux ব্যবহার করলে:**
  ```bash
  GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o attendance-collector main.go
  ```

---

### ১.২ ক্লায়েন্ট ফোল্ডার স্ট্রাকচার

একটি ফোল্ডার তৈরি করুন (যেমন: `AttendanceCollector_Release`) এবং তাতে **শুধুমাত্র** নিচের ২টি ফাইল রাখুন:

```
AttendanceCollector_Release/
├── attendance-collector         (কমপাইল্ড বাইনারি ফাইল — কোড সম্পূর্ণ হিডেন)
└── .env                         (ডিভাইস IP ও ডাটাবেস কনফিগারেশন ফাইল)
```

> 🛑 **মনে রাখবেন**: এই ফোল্ডারে `main.go`, `internal/` ফোল্ডার, `go.mod` বা কোনো সোর্স ফাইল প্রদান করবেন না।

---

## 🚀 ধাপ ২: ক্লায়েন্টের ল্যাপটপে ইনস্টলেশন নির্দেশিকা (Client Installation)

### ২.১ কনফিগারেশন আপডেট (`.env`)

ক্লায়েন্টের ল্যাপটপের IP এবং MySQL/SQLite তথ্যানুযায়ী `.env` ফাইলটি এডিট করুন:

```env
DB_CONNECTION=mysql
DB_HOST=127.0.0.1
DB_PORT=35060
DB_DATABASE=attendance
DB_USERNAME=root
DB_PASSWORD=password
AUTO_MIGRATE=false

ZK_IN_NAME=Outside (Entry)
ZK_IN_IP=192.168.10.201
ZK_IN_PORT=4370

ZK_OUT_NAME=Inside (Exit)
ZK_OUT_IP=192.168.10.55
ZK_OUT_PORT=4370

POLL_INTERVAL_MINUTES=10
STAGGER_MINUTES=5
```

---

### ২.২ সংযোগ পরীক্ষা (Test Connection)

টার্মিনাল/কমান্ড প্রম্পট দিয়ে সংযোগ পরীক্ষা করুন:

- **macOS (Intel Mac & Apple Silicon):**
  ```bash
  cd /path/to/AttendanceCollector_Release
  
  # ১. বাইনারি এক্সিকিউট পারমিশন দিন
  chmod +x attendance-collector
  
  # ২. macOS সিকিউরিটি কোয়ারেন্টাইন বাইপাস (যদি Unidentified Developer ওয়ার্নিং দেখায়)
  xattr -d com.apple.quarantine attendance-collector 2>/dev/null || true
  
  # ৩. টেস্ট চালান
  ./attendance-collector test
  ```

- **Windows:**
  ```cmd
  attendance-collector.exe test
  ```

---

## 🔄 ধাপ ৩: ল্যাপটপ অন হলে অটো-স্টার্ট সেটআপ (Auto-Start Configuration)

### ৩.১ macOS-এ অটো-স্টার্ট সেটআপ (macOS LaunchDaemon)

> ⚠️ **macOS 15 Sequoia নোট:** নতুন ম্যাক-এ "Local Network Privacy" এর কারণে ব্যাকগ্রাউন্ড প্রসেস লোকাল আইপিতে কানেক্ট করতে বাধা পায় (No route to host এরর দেয়)। এটি এড়াতে `LaunchAgent` এর বদলে রুট পারমিশন সহ `LaunchDaemon` ব্যবহার করতে হবে।

১. ক্লায়েন্টের Mac-এ টার্মিনাল খুলে নিচের কমান্ড দিয়ে সার্ভিস ফাইল তৈরি করুন:
   ```bash
   sudo nano /Library/LaunchDaemons/com.attendance.collector.plist
   ```

২. নিচের XML পেস্ট করে ক্লায়েন্টের আসল ডিরেক্টরি পাথ অনুযায়ী পরিবর্তন করুন:
   ```xml
   <?xml version="1.0" encoding="UTF-8"?>
   <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
   <plist version="1.0">
   <dict>
       <key>Label</key>
       <string>com.attendance.collector</string>
       
       <key>ProgramArguments</key>
       <array>
           <string>/Users/client-name/AttendanceCollector_Release/attendance-collector</string>
           <string>run</string>
       </array>
       
       <key>WorkingDirectory</key>
       <string>/Users/client-name/AttendanceCollector_Release</string>
       
       <key>RunAtLoad</key>
       <true/>
       
       <key>KeepAlive</key>
       <true/>
       
       <key>StandardOutPath</key>
       <string>/Users/client-name/AttendanceCollector_Release/daemon.log</string>
       
       <key>StandardErrorPath</key>
       <string>/Users/client-name/AttendanceCollector_Release/daemon.err.log</string>
   </dict>
   </plist>
   ```

৩. সার্ভিস একটিভ (Start) করুন:
   ```bash
   sudo launchctl load /Library/LaunchDaemons/com.attendance.collector.plist
   ```

৪. সার্ভিস চালু হয়েছে কিনা চেক করতে:
   ```bash
   sudo launchctl list | grep com.attendance.collector
   # (0 স্ট্যাটাস দেখালে বুঝতে হবে সার্ভিস ঠিকমতো চলছে)
   ```

৫. ব্যাকগ্রাউন্ড লগ দেখতে (কোড কাজ করছে কিনা):
   ```bash
   tail -n 20 ~/AttendanceCollector_Release/daemon.log
   ```

৬. প্রয়োজনে সার্ভিস বন্ধ (Stop) বা আপডেট করতে চাইলে:
   ```bash
   sudo launchctl unload /Library/LaunchDaemons/com.attendance.collector.plist
   ```
   *(নোট: নতুন কোনো বাইনারি দিলে আগে `unload` করে বাইনারি রিপ্লেস করে তারপর আবার `load` করতে হবে।)*
   
---

### ৩.২ বিকল্প উপায়: macOS Automator App + Login Items (No Sudo / No Terminal)

যদি ক্লায়েন্ট টার্মিনাল বা `sudo` পাসওয়ার্ড ব্যবহার করতে না চান, তবে এটি সবচেয়ে নিরাপদ ও সহজ উপায়। 
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

### ৩.৩ Windows-এ অটো-স্টার্ট সেটআপ (Windows Startup Folder)

১. `Win + R` চেপে টাইপ করুন: `shell:startup` এবং Enter দিন।
২. `attendance-collector.exe` ফাইলটির একটি **Shortcut** তৈরি করে ওই Startup ফোল্ডারে পেস্ট করে দিন।
৩. শর্টকাটের Target-এর শেষে ` run` যোগ করে দিন (যেমন: `C:\Path\attendance-collector.exe run`)।

---

### ৩.৪ Linux-এ অটো-স্টার্ট সেটআপ (Linux Systemd)

১. সার্ভিস ফাইল তৈরি করুন:
   ```bash
   sudo nano /etc/systemd/system/attendance-collector.service
   ```

২. কনফিগারেশন পেস্ট করুন:
   ```ini
   [Unit]
   Description=ZKTeco Attendance Collector
   After=network.target mysql.service

   [Service]
   Type=simple
   User=root
   WorkingDirectory=/path/to/AttendanceCollector_Release
   ExecStart=/path/to/AttendanceCollector_Release/attendance-collector run
   Restart=always
   RestartSec=10
   EnvironmentFile=/path/to/AttendanceCollector_Release/.env

   [Install]
   WantedBy=multi-user.target
   ```

৩. সার্ভিস এনেবল ও চালু করুন:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable attendance-collector
   sudo systemctl start attendance-collector
   ```
