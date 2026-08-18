# ZKTeco Attendance Collector — Linux Run & Systemd Setup

This guide shows how to run the `attendance-collector-linux` binary on Linux and configure it as a `systemd` service.

## 1. Project Location

Current project directory:

```bash
/home/appnapglobal/attendance-scrapper
```

Expected files:

```text
/home/appnapglobal/attendance-scrapper/
├── attendance-collector-linux
├── .env
└── README.md
```

Check the files:

```bash
cd /home/appnapglobal/attendance-scrapper
ls -la
```

---

## 2. Make Sure the Binary Is Executable

Check permission:

```bash
ls -la attendance-collector-linux
```

If it is not executable, run:

```bash
chmod +x attendance-collector-linux
```

---

## 3. Configure the `.env` File

Edit the environment file:

```bash
nano /home/appnapglobal/attendance-scrapper/.env
```

Use normal `KEY=VALUE` format:

```env
DB_HOST=127.0.0.1
DB_PORT=3306
DB_DATABASE=attendance
DB_USERNAME=your_username
DB_PASSWORD=your_password
```

Do not use:

```bash
export DB_HOST=127.0.0.1
```

Save in Nano:

```text
Ctrl + O
Enter
Ctrl + X
```

---

## 4. Test the Application Manually

Go to the project directory:

```bash
cd /home/appnapglobal/attendance-scrapper
```

Run:

```bash
./attendance-collector-linux run
```

If the application starts successfully, stop it with:

```text
Ctrl + C
```

Then configure it as a background service.

---

## 5. Create the Systemd Service

Create the service file:

```bash
sudo nano /etc/systemd/system/attendance-collector.service
```

Add:

```ini
[Unit]
Description=ZKTeco Attendance Collector
After=network-online.target mysql.service
Wants=network-online.target

[Service]
Type=simple
User=appnapglobal
Group=appnapglobal
WorkingDirectory=/home/appnapglobal/attendance-scrapper
EnvironmentFile=/home/appnapglobal/attendance-scrapper/.env

ExecStart=/home/appnapglobal/attendance-scrapper/attendance-collector-linux run

Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Save and exit.

---

## 6. Reload Systemd

After creating or changing the service file:

```bash
sudo systemctl daemon-reload
```

---

## 7. Enable the Service on Server Boot

```bash
sudo systemctl enable attendance-collector
```

This makes the attendance collector start automatically after the server reboots.

---

## 8. Start the Service

```bash
sudo systemctl start attendance-collector
```

---

## 9. Check Service Status

```bash
sudo systemctl status attendance-collector
```

A healthy service should show:

```text
Active: active (running)
```

Press:

```text
q
```

to exit the status screen.

---

## 10. View Live Logs

```bash
sudo journalctl -u attendance-collector -f
```

Stop following logs with:

```text
Ctrl + C
```

---

## 11. View Recent Logs

Last 100 log lines:

```bash
sudo journalctl -u attendance-collector -n 100 --no-pager
```

Logs from the current boot:

```bash
sudo journalctl -u attendance-collector -b --no-pager
```

---

## 12. Restart the Service

After updating the binary or `.env` file:

```bash
sudo systemctl restart attendance-collector
```

Then verify:

```bash
sudo systemctl status attendance-collector
```

---

## 13. Stop the Service

```bash
sudo systemctl stop attendance-collector
```

---

## 14. Start the Service Again

```bash
sudo systemctl start attendance-collector
```

---

## 15. Disable Auto Start

If you do not want the application to automatically start after reboot:

```bash
sudo systemctl disable attendance-collector
```

---

## 16. Updating the Binary

Stop the service first:

```bash
sudo systemctl stop attendance-collector
```

Replace:

```text
/home/appnapglobal/attendance-scrapper/attendance-collector-linux
```

Make sure the new binary is executable:

```bash
chmod +x /home/appnapglobal/attendance-scrapper/attendance-collector-linux
```

Start the service:

```bash
sudo systemctl start attendance-collector
```

Check:

```bash
sudo systemctl status attendance-collector
```

Watch logs:

```bash
sudo journalctl -u attendance-collector -f
```

---

## 17. If the Service Fails

Check status:

```bash
sudo systemctl status attendance-collector
```

Check detailed logs:

```bash
sudo journalctl -u attendance-collector -n 200 --no-pager
```

Test the binary manually:

```bash
cd /home/appnapglobal/attendance-scrapper
./attendance-collector-linux run
```

Check `.env`:

```bash
cat /home/appnapglobal/attendance-scrapper/.env
```

Check binary permissions:

```bash
ls -la /home/appnapglobal/attendance-scrapper/attendance-collector-linux
```

---

## 18. After Changing the Systemd Service File

If you modify:

```text
/etc/systemd/system/attendance-collector.service
```

always run:

```bash
sudo systemctl daemon-reload
sudo systemctl restart attendance-collector
```

---

## 19. Quick Command Reference

```bash
# Start
sudo systemctl start attendance-collector

# Stop
sudo systemctl stop attendance-collector

# Restart
sudo systemctl restart attendance-collector

# Status
sudo systemctl status attendance-collector

# Enable on boot
sudo systemctl enable attendance-collector

# Disable on boot
sudo systemctl disable attendance-collector

# Live logs
sudo journalctl -u attendance-collector -f

# Last 100 logs
sudo journalctl -u attendance-collector -n 100 --no-pager

# Reload systemd after service file changes
sudo systemctl daemon-reload
```

## Final Service Path

```text
/etc/systemd/system/attendance-collector.service
```

## Final Application Path

```text
/home/appnapglobal/attendance-scrapper
```

## Final Executable Path

```text
/home/appnapglobal/attendance-scrapper/attendance-collector-linux
```
