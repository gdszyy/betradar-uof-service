#!/usr/bin/env python3
"""
完整的 Chip ID 分配脚本
使用高效的 SQL 逻辑为所有市场分配 chip_id
"""

import psycopg2
import sys
import os
from datetime import datetime
import re

def get_db_connection():
    """获取数据库连接"""
    db_url = os.environ.get('DATABASE_URL')
    if not db_url:
        print("错误：未设置 DATABASE_URL 环境变量")
        sys.exit(1)
    
    try:
        conn = psycopg2.connect(db_url)
        return conn
    except Exception as e:
        print(f"错误：无法连接到数据库: {e}")
        sys.exit(1)

def parse_specifiers(specifiers_str):
    """解析 specifiers 字符串为字典"""
    if not specifiers_str:
        return {}
    
    specifiers = {}
    parts = specifiers_str.split('|')
    for part in parts:
        if '=' in part:
            key, value = part.split('=', 1)
            specifiers[key.strip()] = value.strip()
    
    return specifiers

def assign_all_chip_ids():
    """运行完整的 Chip ID 分配"""
    conn = get_db_connection()
    cursor = conn.cursor()
    
    print("=" * 80)
    print("完整的 Chip ID 分配")
    print("=" * 80)
    print(f"开始时间：{datetime.now()}\n")
    
    try:
        # 1. 检查初始状态
        print("【步骤 1】检查初始状态...")
        cursor.execute("""
            SELECT 
                COUNT(*) as total,
                COUNT(CASE WHEN chip_id IS NOT NULL AND chip_id != '' THEN 1 END) as chip_mapped,
                COUNT(CASE WHEN tab_id IS NOT NULL AND tab_id != '' THEN 1 END) as tab_mapped
            FROM markets
        """)
        total, chip_mapped, tab_mapped = cursor.fetchone()
        print(f"  总市场数：{total:,}")
        print(f"  已分配 chip_id：{chip_mapped:,} ({(chip_mapped/total)*100:.2f}%)")
        print(f"  已分配 tab_id：{tab_mapped:,} ({(tab_mapped/total)*100:.2f}%)\n")
        
        # 2. 获取所有需要分配的市场和 chips 信息
        print("【步骤 2】加载 Chips 配置...")
        
        # 获取所有 chips
        cursor.execute("""
            SELECT id, tab_id, specifier, value
            FROM market_chips
            WHERE specifier IS NOT NULL AND value IS NOT NULL
            ORDER BY tab_id, specifier, value
        """)
        
        chips = cursor.fetchall()
        print(f"  加载了 {len(chips)} 个 chips\n")
        
        # 3. 为市场分配 chip_id
        print("【步骤 3】分配 Chip ID...")
        
        total_assigned = 0
        batch_size = 1000
        updates = []
        
        # 获取所有需要分配的市场
        cursor.execute("""
            SELECT id, tab_id, specifiers
            FROM markets
            WHERE (chip_id IS NULL OR chip_id = '')
            AND tab_id IS NOT NULL AND tab_id != ''
            ORDER BY id
        """)
        
        markets = cursor.fetchall()
        
        for market_id, tab_id, specifiers_str in markets:
            specifiers = parse_specifiers(specifiers_str)
            
            # 查找匹配的 chip
            chip_id = None
            for chip_id_val, chip_tab_id, specifier, value in chips:
                if chip_tab_id == tab_id and specifier in specifiers:
                    if specifiers[specifier] == value:
                        chip_id = chip_id_val
                        break
            
            if chip_id:
                updates.append((chip_id, market_id))
                total_assigned += 1
            
            # 批量更新
            if len(updates) >= batch_size:
                for chip_id_val, market_id_val in updates:
                    cursor.execute("""
                        UPDATE markets
                        SET chip_id = %s, updated_at = CURRENT_TIMESTAMP
                        WHERE id = %s
                    """, (chip_id_val, market_id_val))
                conn.commit()
                print(f"  已处理 {total_assigned:,} 个市场...")
                updates = []
        
        # 处理剩余的批次
        if updates:
            for chip_id_val, market_id_val in updates:
                cursor.execute("""
                    UPDATE markets
                    SET chip_id = %s, updated_at = CURRENT_TIMESTAMP
                    WHERE id = %s
                """, (chip_id_val, market_id_val))
            conn.commit()
        
        print(f"  ✓ 分配了 {total_assigned:,} 个市场的 chip_id\n")
        
        # 4. 验证结果
        print("【步骤 4】验证分配结果...")
        cursor.execute("""
            SELECT 
                COUNT(*) as total,
                COUNT(CASE WHEN chip_id IS NOT NULL AND chip_id != '' THEN 1 END) as chip_mapped,
                COUNT(CASE WHEN chip_id IS NULL OR chip_id = '' THEN 1 END) as chip_unmapped
            FROM markets
        """)
        total_final, chip_mapped_final, chip_unmapped_final = cursor.fetchone()
        print(f"  总市场数：{total_final:,}")
        print(f"  已分配 chip_id：{chip_mapped_final:,} ({(chip_mapped_final/total_final)*100:.2f}%)")
        print(f"  未分配 chip_id：{chip_unmapped_final:,}\n")
        
        # 5. 显示 Tab + Chip 的组合分布
        print("【步骤 5】Tab + Chip 组合分布...")
        cursor.execute("""
            SELECT tab_id, chip_id, COUNT(*) as count
            FROM markets
            WHERE tab_id IS NOT NULL
            GROUP BY tab_id, chip_id
            ORDER BY tab_id, count DESC
        """)
        
        print("  Tab ID          | Chip ID                    | 市场数")
        print("  " + "-" * 70)
        for tab_id, chip_id, count in cursor.fetchall():
            chip_str = chip_id if chip_id else "NULL"
            print(f"  {tab_id:15} | {chip_str:26} | {count:,}")
        
        print(f"\n✓ 分配完成！")
        print(f"结束时间：{datetime.now()}")
        
    except Exception as e:
        print(f"✗ 错误：{e}")
        import traceback
        traceback.print_exc()
        conn.rollback()
        sys.exit(1)
    finally:
        cursor.close()
        conn.close()

if __name__ == '__main__':
    assign_all_chip_ids()
