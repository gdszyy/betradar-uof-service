#!/usr/bin/env python3
"""
市场 Chip ID 分配脚本
基于市场的 specifiers 和 tab_id，为市场分配对应的 chip_id
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
    """
    解析 specifiers 字符串为字典
    格式：key1=value1|key2=value2
    """
    if not specifiers_str:
        return {}
    
    specifiers = {}
    # 按 | 分割
    parts = specifiers_str.split('|')
    for part in parts:
        if '=' in part:
            key, value = part.split('=', 1)
            specifiers[key.strip()] = value.strip()
    
    return specifiers

def assign_chip_ids():
    """运行完整的 Chip ID 分配"""
    conn = get_db_connection()
    cursor = conn.cursor()
    
    print("=" * 60)
    print("市场 Chip ID 分配")
    print("=" * 60)
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
        print(f"  已分配 chip_id：{chip_mapped:,}")
        print(f"  已分配 tab_id：{tab_mapped:,}\n")
        
        # 2. 获取所有需要分配 chip_id 的市场
        print("【步骤 2】获取需要分配的市场...")
        cursor.execute("""
            SELECT id, tab_id, specifiers
            FROM markets
            WHERE (chip_id IS NULL OR chip_id = '')
            AND tab_id IS NOT NULL AND tab_id != ''
            AND specifiers IS NOT NULL AND specifiers != ''
            ORDER BY id
        """)
        
        markets = cursor.fetchall()
        print(f"  需要分配 chip_id 的市场：{len(markets):,}\n")
        
        # 3. 为每个市场分配 chip_id
        print("【步骤 3】分配 chip_id...")
        
        assigned_count = 0
        unmatched_count = 0
        batch_size = 1000
        updates = []
        
        for market_id, tab_id, specifiers_str in markets:
            # 解析 specifiers
            specifiers = parse_specifiers(specifiers_str)
            
            # 尝试找到匹配的 chip
            chip_id = None
            
            # 按优先级尝试匹配 specifier
            # 优先级：quarternr, setnr, mapnr, inningnr, periodnr, framenr, overnr, drivenr, 其他
            priority_specifiers = [
                'quarternr', 'setnr', 'mapnr', 'inningnr', 
                'periodnr', 'framenr', 'overnr', 'drivenr',
                'goalnr', 'roundnr', 'gamenr', 'cornernr',
                'pointnr', 'pitchnr', 'count', 'dismissalnr',
                'deliverynr', 'legnr', 'endnr', 'playnr'
            ]
            
            for spec_name in priority_specifiers:
                if spec_name in specifiers:
                    spec_value = specifiers[spec_name]
                    
                    # 查询匹配的 chip
                    cursor.execute("""
                        SELECT id FROM market_chips
                        WHERE tab_id = %s
                        AND specifier = %s
                        AND value = %s
                        LIMIT 1
                    """, (tab_id, spec_name, spec_value))
                    
                    result = cursor.fetchone()
                    if result:
                        chip_id = result[0]
                        break
            
            if chip_id:
                updates.append((chip_id, market_id))
                assigned_count += 1
            else:
                unmatched_count += 1
            
            # 批量更新
            if len(updates) >= batch_size:
                for chip_id, market_id in updates:
                    cursor.execute("""
                        UPDATE markets
                        SET chip_id = %s, updated_at = CURRENT_TIMESTAMP
                        WHERE id = %s
                    """, (chip_id, market_id))
                conn.commit()
                print(f"  已处理 {assigned_count + unmatched_count:,} 个市场...")
                updates = []
        
        # 处理剩余的批次
        if updates:
            for chip_id, market_id in updates:
                cursor.execute("""
                    UPDATE markets
                    SET chip_id = %s, updated_at = CURRENT_TIMESTAMP
                    WHERE id = %s
                """, (chip_id, market_id))
            conn.commit()
        
        print(f"  ✓ 已分配 {assigned_count:,} 个市场的 chip_id")
        print(f"  ⚠ 未匹配 {unmatched_count:,} 个市场\n")
        
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
        
        # 5. 显示 chip 分布
        print("【步骤 5】Chip 分布统计（前 20）...")
        cursor.execute("""
            SELECT chip_id, COUNT(*) as count
            FROM markets
            WHERE chip_id IS NOT NULL AND chip_id != ''
            GROUP BY chip_id
            ORDER BY count DESC
            LIMIT 20
        """)
        
        print("  Chip ID                        | 市场数")
        print("  " + "-" * 50)
        for chip_id, count in cursor.fetchall():
            print(f"  {chip_id:30} | {count:,}")
        
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
    assign_chip_ids()
