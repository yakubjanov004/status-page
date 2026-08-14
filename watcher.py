import sys
import time
import subprocess
import datetime

def get_service_state(service_name):
    try:
        # systemctl is-active qaytaradigan qiymatlar: 'active', 'inactive', 'failed', 'activating' va hk.
        result = subprocess.run(
            ['systemctl', 'is-active', service_name],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        return result.stdout.strip()
    except Exception as e:
        return f"error: {e}"

def main():
    if len(sys.argv) < 2:
        print("Foydalanish: python3 watcher.py <xizmat-nomi.service>")
        print("Masalan: python3 watcher.py odimrepo-frontend.service")
        sys.exit(1)

    service_name = sys.argv[1]
    
    print(f"[{datetime.datetime.now().strftime('%H:%M:%S')}] 👀 '{service_name}' xizmati kuzatilmoqda...")
    print("Test qilish uchun boshqa terminaldan 'systemctl stop/start/restart' berib ko'ring.")
    print("Kuzatishni to'xtatish va statistikani ko'rish uchun CTRL+C bosing.\n")

    current_state = get_service_state(service_name)
    print(f"-> Boshlang'ich holat: {current_state.upper()}")

    stats = {
        'total_downs': 0,
        'total_recoveries': 0,
        'downtimes': [], # uzilish davomiyliklari (soniyalarda)
    }

    start_time = time.time()
    last_down_time = None

    try:
        while True:
            new_state = get_service_state(service_name)
            
            # Agar holat o'zgargan bo'lsa
            if new_state != current_state:
                now_str = datetime.datetime.now().strftime('%H:%M:%S.%f')[:-3]
                
                # Active dan boshqa holatga o'tdi (o'chdi)
                if current_state == 'active' and new_state != 'active':
                    print(f"[{now_str}] 🔴 XIZMAT O'CHDI! Holat: {new_state.upper()}")
                    stats['total_downs'] += 1
                    last_down_time = time.time()
                
                # Boshqa holatdan Active ga o'tdi (yondi)
                elif current_state != 'active' and new_state == 'active':
                    duration = 0
                    if last_down_time:
                        duration = time.time() - last_down_time
                        stats['downtimes'].append(duration)
                    
                    dur_str = f"{duration:.3f} soniya" if duration else "Noma'lum"
                    print(f"[{now_str}] 🟢 XIZMAT TIKLANDI! Holat: ACTIVE (Qayta ishlashga ketgan vaqt: {dur_str})")
                    stats['total_recoveries'] += 1
                    last_down_time = None
                
                # Boshqa oraliq holatlar (masalan, deactivating -> inactive)
                else:
                    print(f"[{now_str}] ⚠️ Holat o'zgardi: {current_state.upper()} -> {new_state.upper()}")

                current_state = new_state
            
            # Juda tezkor reaksiya uchun 0.1 soniya kutamiz
            time.sleep(0.1)

    except KeyboardInterrupt:
        total_time = time.time() - start_time
        print("\n\n" + "="*55)
        print("📊 UMUMIY STATISTIKA (Kuzatuv yakunlandi)")
        print("="*55)
        print(f"Kuzatilgan xizmat:   {service_name}")
        print(f"Jami kuzatuv vaqti:  {total_time:.2f} soniya")
        print(f"Necha marta o'chdi:  {stats['total_downs']} marta")
        print(f"Necha marta yondi:   {stats['total_recoveries']} marta")
        
        if stats['downtimes']:
            avg_downtime = sum(stats['downtimes']) / len(stats['downtimes'])
            max_downtime = max(stats['downtimes'])
            total_downtime = sum(stats['downtimes'])
            print(f"Jami o'chik vaqt:    {total_downtime:.3f} soniya")
            print(f"O'rtacha tiklanish:  {avg_downtime:.3f} soniya (downtime)")
            print(f"Eng uzoq tiklanish:  {max_downtime:.3f} soniya")
        else:
            print("Jami o'chik vaqt:    0 soniya (hech qanday uzilish bo'lmadi)")
        print("="*55)
        sys.exit(0)

if __name__ == "__main__":
    main()
