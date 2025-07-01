package db

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/paragor/todo/pkg/models"

	pgxUUID "github.com/vgarvardt/pgx-google-uuid/v5"
)

const postgresqlCurrentVersion = 1

type postgresqlTasksRepository struct {
	conn         *pgxpool.Pool
	connString   string
	readTimeout  time.Duration
	writeTimeout time.Duration

	wg sync.WaitGroup

	ctx    context.Context
	cancel func()
}

func NewPostgresqlTasksRepository(connString string) *postgresqlTasksRepository {
	return &postgresqlTasksRepository{
		readTimeout:  time.Second * 5,
		writeTimeout: time.Second * 5,
		connString:   connString,
	}
}

func (r *postgresqlTasksRepository) Get(UUID uuid.UUID) (*models.Task, error) {
	if r.ctx == nil {
		return nil, fmt.Errorf("repository is not started")
	}
	select {
	case <-r.ctx.Done():
		return nil, fmt.Errorf("repository is closed")
	default:
		r.wg.Add(1)
		defer r.wg.Done()
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.readTimeout)
	defer cancel()

	task := &models.Task{}
	err := r.conn.QueryRow(
		ctx,
		"SELECT task_data FROM tasks WHERE uuid::uuid = $1::uuid LIMIT 1",
		UUID,
	).Scan(task)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("error on get task from database: %w", err)

	}
	return task, nil
}

func (r *postgresqlTasksRepository) Insert(task *models.Task) error {
	if r.ctx == nil {
		return fmt.Errorf("repository is not started")
	}
	select {
	case <-r.ctx.Done():
		return fmt.Errorf("repository is closed")
	default:
		r.wg.Add(1)
		defer r.wg.Done()
	}
	ctx, cancel := context.WithTimeout(context.Background(), r.writeTimeout)
	defer cancel()

	task.Unify()
	if err := task.Validate(); err != nil {
		return fmt.Errorf("invalid task: %w", err)
	}

	// 	if _, err := db.Exec(context.Background(), `insert into shortened_urls(id, url) values ($1, $2)
	//	on conflict (id) do update set url=excluded.url`, id, url); err == nil {
	_, err := r.conn.Exec(ctx, `
INSERT INTO
	tasks(uuid, version, task_data)
	values ($1, $2, $3)
ON CONFLICT (uuid) 
DO UPDATE 
SET version = EXCLUDED.version, task_data = EXCLUDED.task_data
`, task.UUID, postgresqlCurrentVersion, task)
	if err != nil {
		return fmt.Errorf("error on insert task into postgresql: %w", err)
	}

	return nil
}

func (r *postgresqlTasksRepository) All() ([]*models.Task, error) {
	if r.ctx == nil {
		return nil, fmt.Errorf("repository is not started")
	}
	select {
	case <-r.ctx.Done():
		return nil, fmt.Errorf("repository is closed")
	default:
		r.wg.Add(1)
		defer r.wg.Done()
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.readTimeout)
	defer cancel()
	result := []*models.Task{}
	rows, err := r.conn.Query(ctx, "SELECT task_data FROM tasks")
	if err != nil {
		return nil, fmt.Errorf("error on list tasks from postgresql: %w", err)
	}
	for rows.Next() {
		task := &models.Task{}
		err := rows.Scan(task)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("error on get another task from postgresql: %w", err)
		}
		result = append(result, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error after list tasks from postgresql: %w", err)
	}
	rows.Close()
	return result, nil
}

func (r *postgresqlTasksRepository) Stop() {
	r.wg.Wait()
	if r.conn != nil {
		r.conn.Close()
	}
}

//go:embed postgresql_migrations/*
var migrationsFS embed.FS

func (r *postgresqlTasksRepository) Start(ctx context.Context, stopper chan<- error) error {
	migrationsSource, err := iofs.New(migrationsFS, "postgresql_migrations")
	if err != nil {
		return fmt.Errorf("error on read migrations postgresql: %w", err)
	}

	config, err := pgxpool.ParseConfig(r.connString)
	if err != nil {
		return fmt.Errorf("error on parse config of postgresql: %w", err)
	}
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		pgxUUID.Register(conn.TypeMap())
		return nil
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("error on connection to postgresql: %w", err)
	}
	{
		pingCtx, pingCtxCancel := context.WithTimeout(ctx, min(r.writeTimeout, r.readTimeout))
		defer pingCtxCancel()
		if err := pool.Ping(pingCtx); err != nil {
			return fmt.Errorf("cant ping postgresql: %w", err)
		}
	}
	migrations, err := migrate.NewWithSourceInstance(
		"iofs",
		migrationsSource,
		r.connString,
	)
	if err != nil {
		return fmt.Errorf("error on create migrations instance of postgresql: %w", err)
	}
	defer migrations.Close()

	log.Println("start postgresql migrations")
	if err := migrations.Up(); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("error on run migrations postgresql: %w", err)
		}
		log.Println(err.Error())
	}
	log.Println("finish postgresql migrations")

	r.conn = pool
	r.ctx, r.cancel = context.WithCancel(ctx)
	go func() {
		select {
		case <-r.ctx.Done():
			err := r.ctx.Err()
			stopper <- fmt.Errorf("stop db.postgresql: %w", err)
		}
	}()

	return nil
}
