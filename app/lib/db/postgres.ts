import { Pool, PoolConfig, QueryResultRow, PoolClient, QueryResult } from "pg";

type Primitive = string | number | boolean | undefined | null;

class DBClient {
  client: PoolClient
  constructor(client: PoolClient) {
    this.client = client;
  }

  /**
   * A template literal tag providing safe, easy to use SQL parameterization.
   * Parameters are substituted using the underlying Postgres database, and so must follow
   * the rules of Postgres parameterization.
   * @example
   * ```ts
   * const pool = createPool();
   * const userId = 123;
   * const result = await pool.sql`SELECT * FROM users WHERE id = ${userId}`;
   * // Equivalent to: await pool.query('SELECT * FROM users WHERE id = $1', [id]);
   * ```
   * @returns A promise that resolves to the query result.
   */
  async sql<O extends QueryResultRow>(strings: TemplateStringsArray, ...values: Primitive[]): Promise<QueryResult<O>> {
    const sqlString = strings.reduce(
      (acc, str, i) => acc + (i === 0 ? str : `$${i} ` + str),
      ''
    );

    try {
      const res: QueryResult<O> = await this.client.query({ text: sqlString, values });
      return res as QueryResult<O>;
    } catch (error) {
      console.error('Error executing query:', error);
      throw error;
    }
  }

  release(err?: Error | boolean): void {
    this.client.release(err)
  }
}

class DBPool {
  pool: Pool
  constructor(config?: PoolConfig) {
    this.pool = new Pool(config);
  }

  /**
   * A template literal tag providing safe, easy to use SQL parameterization.
   * Parameters are substituted using the underlying Postgres database, and so must follow
   * the rules of Postgres parameterization.
   * @example
   * ```ts
   * const pool = createPool();
   * const userId = 123;
   * const result = await pool.sql`SELECT * FROM users WHERE id = ${userId}`;
   * // Equivalent to: await pool.query('SELECT * FROM users WHERE id = $1', [id]);
   * ```
   * @returns A promise that resolves to the query result.
   */
  async sql<O extends QueryResultRow>(strings: TemplateStringsArray, ...values: Primitive[]): Promise<QueryResult<O>> {
    const sqlString = strings.reduce(
      (acc, str, i) => acc + (i === 0 ? str : `$${i} ` + str),
      ''
    );

    try {
      const res: QueryResult<O> = await this.pool.query({ text: sqlString, values });
      return res as QueryResult<O>;
    } catch (error) {
      console.error('Error executing query:', error);
      throw error;
    }
  }

  connect(): Promise<DBClient>;
  // connect(callback: (err: Error | undefined, client: PoolClient | undefined, done: (release?: any) => void) => void): void;

  async connect(): Promise<DBClient> {
    return new DBClient(await this.pool.connect())
  }

  // connect(callback?: (err: Error | undefined, client: PoolClient | undefined, done: (release?: any) => void) => void): void {
  //   return this.pool.connect(callback!)
  // }
}

const config = {
  user: process.env.POSTGRES_USER,
  password: process.env.POSTGRES_PASSWORD,
  host: process.env.POSTGRES_HOST,
  port: Number.parseInt(process.env.PGSQL_PORT as string),
  database: process.env.POSTGRES_DATABASE,
};

const db = new DBPool(config);
const client = db;

async function sql<O extends QueryResultRow>(strings: TemplateStringsArray, ...values: Primitive[]): Promise<QueryResult<O>> {
  return client.sql(strings, ...values)
}

export { db, sql, client };
