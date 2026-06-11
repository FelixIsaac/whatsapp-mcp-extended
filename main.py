import os
import sqlite3
import json

def manage_whitelist(args):
    db = sqlite3.connect('whatsapp.db')
    cursor = db.cursor()

    if args.action == 'add':
        cursor.execute("INSERT INTO allowed_jids (jid, label) VALUES (?, ?)", (args.jid, args.label))
        db.commit()
        print(f"Added {args.jid} to whitelist")
    elif args.action == 'remove':
        cursor.execute("DELETE FROM allowed_jids WHERE jid = ?", (args.jid,))
        db.commit()
        print(f"Removed {args.jid} from whitelist")
    elif args.action == 'list':
        cursor.execute("SELECT jid, label FROM allowed_jids")
        rows = cursor.fetchall()
        for row in rows:
            print(f"{row[0]} - {row[1]}")

    db.close()

if __name__ == '__main__':
    import argparse
    parser = argparse.ArgumentParser(description='Manage whitelist')
    subparsers = parser.add_subparsers(dest='action')

    add_parser = subparsers.add_parser('add')
    add_parser.add_argument('--jid', required=True)
    add_parser.add_argument('--label', required=True)

    remove_parser = subparsers.add_parser('remove')
    remove_parser.add_argument('--jid', required=True)

    list_parser = subparsers.add_parser('list')

    args = parser.parse_args()

    if args.action == 'add':
        manage_whitelist(args)
    elif args.action == 'remove':
        manage_whitelist(args)
    elif args.action == 'list':
        manage_whitelist(args)
    else:
        parser.print_help()