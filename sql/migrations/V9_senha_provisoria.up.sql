-- ADR-0020: o admin redefine a senha com uma senha provisória, que o usuário troca no primeiro acesso.
-- A senha continua guardada só como hash (bcrypt): ninguém consegue ver a senha cadastrada.
ALTER TABLE users
    ADD COLUMN must_change_password BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN password_reset_at    TIMESTAMPTZ;
