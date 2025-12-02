#!/usr/bin/env python3
"""
市场 Tab/Chip 完整分配脚本
使用单条 SQL 语句避免死锁
"""

import psycopg2
import sys
import os
from datetime import datetime

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

def run_full_assignment():
    """运行完整的 Tab/Chip 分配"""
    conn = get_db_connection()
    cursor = conn.cursor()
    
    print("=" * 60)
    print("市场 Tab/Chip 完整分配")
    print("=" * 60)
    print(f"开始时间：{datetime.now()}\n")
    
    try:
        # 1. 检查初始状态
        print("【步骤 1】检查初始状态...")
        cursor.execute("""
            SELECT 
                COUNT(*) as total,
                COUNT(CASE WHEN tab_id IS NOT NULL AND tab_id != '' THEN 1 END) as mapped,
                COUNT(CASE WHEN tab_id IS NULL OR tab_id = '' THEN 1 END) as unmapped
            FROM markets
        """)
        total, mapped, unmapped = cursor.fetchone()
        print(f"  总市场数：{total:,}")
        print(f"  已映射：{mapped:,}")
        print(f"  未映射：{unmapped:,}\n")
        
        if unmapped == 0:
            print("✓ 所有市场已映射，无需分配")
            return
        
        # 2. 使用单条 SQL 语句进行完整分配
        print("【步骤 2】执行完整分配（使用 CASE 语句）...")
        
        assignment_sql = """
        UPDATE markets
        SET tab_id = CASE
            -- 基于 market_type 的映射
            WHEN market_type IN ('regular_play', 'player_props', 'micro_market', 'bookings', 
                                'corners', '1st_half', 'combo', '2nd_half', 'scorers')
            THEN market_type
            
            -- 基于 specifiers 的映射
            WHEN specifiers LIKE '%inningnr%' THEN 'innings'
            WHEN specifiers LIKE '%setnr%' THEN 'sets'
            WHEN specifiers LIKE '%mapnr%' THEN 'maps'
            WHEN specifiers LIKE '%quarternr%' THEN 'quarters'
            WHEN specifiers LIKE '%periodnr%' THEN 'periods'
            WHEN specifiers LIKE '%framenr%' THEN 'frames'
            WHEN specifiers LIKE '%overnr%' THEN 'overs'
            WHEN specifiers LIKE '%drivenr%' THEN 'drives'
            
            -- 默认值
            ELSE 'regular_play'
        END,
        updated_at = CURRENT_TIMESTAMP
        WHERE tab_id IS NULL OR tab_id = ''
        """
        
        cursor.execute(assignment_sql)
        count = cursor.rowcount
        conn.commit()
        
        print(f"  ✓ 已分配 {count:,} 个市场\n")
        
        # 3. 验证结果
        print("【步骤 3】验证分配结果...")
        cursor.execute("""
            SELECT 
                COUNT(*) as total,
                COUNT(CASE WHEN tab_id IS NOT NULL AND tab_id != '' THEN 1 END) as mapped,
                COUNT(CASE WHEN tab_id IS NULL OR tab_id = '' THEN 1 END) as unmapped
            FROM markets
        """)
        total_final, mapped_final, unmapped_final = cursor.fetchone()
        print(f"  总市场数：{total_final:,}")
        print(f"  已映射：{mapped_final:,}")
        print(f"  未映射：{unmapped_final:,}")
        print(f"  映射率：{(mapped_final/total_final)*100:.2f}%\n")
        
        # 4. 显示 Tab 分布
        print("【步骤 4】Tab 分布统计...")
        cursor.execute("""
            SELECT tab_id, COUNT(*) as count
            FROM markets
            WHERE tab_id IS NOT NULL
            GROUP BY tab_id
            ORDER BY count DESC
        """)
        
        print("  Tab ID          | 市场数")
        print("  " + "-" * 40)
        for tab_id, count in cursor.fetchall():
            print(f"  {tab_id:15} | {count:,}")
        
        print(f"\n✓ 分配完成！")
        print(f"结束时间：{datetime.now()}")
        
    except Exception as e:
        print(f"✗ 错误：{e}")
        conn.rollback()
        sys.exit(1)
    finally:
        cursor.close()
        conn.close()

if __name__ == '__main__':
    run_full_assignment()
