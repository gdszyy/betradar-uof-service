#!/usr/bin/env python3
"""
市场 Groups 和 Tabs 完整分配脚本
从 market_descriptions 同步 groups，然后基于 groups 和 specifiers 分配 tabs
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
    """运行完整的 Groups 和 Tabs 分配"""
    conn = get_db_connection()
    cursor = conn.cursor()
    
    print("=" * 60)
    print("市场 Groups 和 Tabs 完整分配")
    print("=" * 60)
    print(f"开始时间：{datetime.now()}\n")
    
    try:
        # 1. 检查初始状态
        print("【初始状态】")
        cursor.execute("""
            SELECT 
                COUNT(*) as total,
                COUNT(CASE WHEN tab_id IS NOT NULL AND tab_id != '' THEN 1 END) as mapped,
                COUNT(CASE WHEN tab_id IS NULL OR tab_id = '' THEN 1 END) as unmapped,
                COUNT(CASE WHEN groups IS NOT NULL AND groups != '' THEN 1 END) as with_groups
            FROM markets
        """)
        total, mapped, unmapped, with_groups = cursor.fetchone()
        print(f"  总市场数：{total:,}")
        print(f"  已分配：{mapped:,}")
        print(f"  未分配：{unmapped:,}")
        print(f"  有 groups：{with_groups:,}\n")
        
        # 2. 同步 groups 从 market_descriptions
        print("【步骤 1】从 market_descriptions 同步 groups...")
        
        # 检查 market_descriptions 表是否有数据
        cursor.execute("SELECT COUNT(*) FROM market_descriptions")
        desc_count = cursor.fetchone()[0]
        print(f"  market_descriptions 表中有 {desc_count:,} 条记录")
        
        if desc_count > 0:
            cursor.execute("""
                UPDATE markets m
                SET groups = md.groups, updated_at = CURRENT_TIMESTAMP
                FROM market_descriptions md
                WHERE m.sr_market_id = md.market_id
                AND (m.groups IS NULL OR m.groups = '')
                AND md.groups IS NOT NULL AND md.groups != ''
            """)
            count = cursor.rowcount
            conn.commit()
            print(f"  ✓ 成功同步 {count:,} 个市场的 groups\n")
        else:
            print(f"  警告：market_descriptions 表为空，跳过同步\n")
        
        # 3. 基于 groups 分配 tabs
        print("【步骤 2】基于 groups 分配 tabs...")
        cursor.execute("""
            UPDATE markets
            SET tab_id = CASE
                WHEN groups LIKE '%regular_play%' THEN 'regular_play'
                WHEN groups LIKE '%player_props%' THEN 'player_props'
                WHEN groups LIKE '%micro_market%' THEN 'micro_market'
                WHEN groups LIKE '%bookings%' THEN 'bookings'
                WHEN groups LIKE '%corners%' THEN 'corners'
                WHEN groups LIKE '%1st_half%' THEN '1st_half'
                WHEN groups LIKE '%combo%' THEN 'combo'
                WHEN groups LIKE '%2nd_half%' THEN '2nd_half'
                WHEN groups LIKE '%scorers%' THEN 'scorers'
                WHEN groups LIKE '%innings%' THEN 'innings'
                WHEN groups LIKE '%sets%' THEN 'sets'
                WHEN groups LIKE '%maps%' THEN 'maps'
                WHEN groups LIKE '%quarters%' THEN 'quarters'
                WHEN groups LIKE '%periods%' THEN 'periods'
                WHEN groups LIKE '%frames%' THEN 'frames'
                WHEN groups LIKE '%overs%' THEN 'overs'
                WHEN groups LIKE '%drives%' THEN 'drives'
            END,
            updated_at = CURRENT_TIMESTAMP
            WHERE (tab_id IS NULL OR tab_id = '')
            AND groups IS NOT NULL AND groups != ''
        """)
        count = cursor.rowcount
        conn.commit()
        print(f"  ✓ 基于 groups 分配了 {count:,} 个市场的 tabs\n")
        
        # 4. 基于 specifiers 分配 tabs
        print("【步骤 3】基于 specifiers 分配 tabs...")
        cursor.execute("""
            UPDATE markets
            SET tab_id = CASE
                WHEN specifiers LIKE '%inningnr%' THEN 'innings'
                WHEN specifiers LIKE '%setnr%' THEN 'sets'
                WHEN specifiers LIKE '%mapnr%' THEN 'maps'
                WHEN specifiers LIKE '%quarternr%' THEN 'quarters'
                WHEN specifiers LIKE '%periodnr%' THEN 'periods'
                WHEN specifiers LIKE '%framenr%' THEN 'frames'
                WHEN specifiers LIKE '%overnr%' THEN 'overs'
                WHEN specifiers LIKE '%drivenr%' THEN 'drives'
            END,
            updated_at = CURRENT_TIMESTAMP
            WHERE (tab_id IS NULL OR tab_id = '')
            AND specifiers IS NOT NULL AND specifiers != ''
        """)
        count = cursor.rowcount
        conn.commit()
        print(f"  ✓ 基于 specifiers 分配了 {count:,} 个市场的 tabs\n")
        
        # 5. 分配默认 tab
        print("【步骤 4】分配默认 tab: regular_play...")
        cursor.execute("""
            UPDATE markets
            SET tab_id = 'regular_play', updated_at = CURRENT_TIMESTAMP
            WHERE tab_id IS NULL OR tab_id = ''
        """)
        count = cursor.rowcount
        conn.commit()
        print(f"  ✓ 分配了 {count:,} 个市场的默认 tab\n")
        
        # 6. 验证结果
        print("【最终统计】")
        cursor.execute("""
            SELECT 
                COUNT(*) as total,
                COUNT(CASE WHEN tab_id IS NOT NULL AND tab_id != '' THEN 1 END) as mapped,
                COUNT(CASE WHEN tab_id IS NULL OR tab_id = '' THEN 1 END) as unmapped,
                COUNT(CASE WHEN groups IS NOT NULL AND groups != '' THEN 1 END) as with_groups
            FROM markets
        """)
        total_final, mapped_final, unmapped_final, with_groups_final = cursor.fetchone()
        print(f"  总市场数：{total_final:,}")
        print(f"  已分配：{mapped_final:,}")
        print(f"  未分配：{unmapped_final:,}")
        print(f"  有 groups：{with_groups_final:,}")
        print(f"  映射率：{(mapped_final/total_final)*100:.2f}%\n")
        
        # 7. 显示 Tab 分布
        print("【Tab 分布】")
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
        
        # 8. 显示 groups 分布（如果有数据）
        if with_groups_final > 0:
            print("\n【Groups 分布（前 10）】")
            cursor.execute("""
                SELECT groups, COUNT(*) as count
                FROM markets
                WHERE groups IS NOT NULL AND groups != ''
                GROUP BY groups
                ORDER BY count DESC
                LIMIT 10
            """)
            
            print("  Groups                      | 市场数")
            print("  " + "-" * 50)
            for groups, count in cursor.fetchall():
                groups_str = groups[:30] if groups else "NULL"
                print(f"  {groups_str:30} | {count:,}")
        
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
