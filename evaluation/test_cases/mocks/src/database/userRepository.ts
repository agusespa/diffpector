import { Pool, PoolClient } from 'pg';
import { User, CreateUserData } from '../types/user';

export class UserRepository {
  private pool: Pool;

  constructor(pool: Pool) {
    this.pool = pool;
  }

  async findById(id: number): Promise<User | null> {
    const client: PoolClient = await this.pool.connect();
    try {
      const query = 'SELECT * FROM users WHERE id = $1';
      const result = await client.query(query, [id]);
      
      if (result.rows.length === 0) {
        return null;
      }
      
      return this.mapRowToUser(result.rows[0]);
    } finally {
      client.release();
    }
  }

  async findByEmail(email: string): Promise<User | null> {
    const client: PoolClient = await this.pool.connect();
    try {
      const query = 'SELECT * FROM users WHERE email = $1';
      const result = await client.query(query, [email]);
      
      if (result.rows.length === 0) {
        return null;
      }
      
      return this.mapRowToUser(result.rows[0]);
    } finally {
      client.release();
    }
  }

  async create(userData: CreateUserData): Promise<User> {
    const client: PoolClient = await this.pool.connect();
    try {
      const query = `
        INSERT INTO users (name, email, created_at) 
        VALUES ($1, $2, NOW()) 
        RETURNING *
      `;
      const result = await client.query(query, [userData.name, userData.email]);
      
      return this.mapRowToUser(result.rows[0]);
    } finally {
      client.release();
    }
  }

  private mapRowToUser(row: any): User {
    return {
      id: row.id,
      name: row.name,
      email: row.email,
      createdAt: row.created_at
    };
  }
}