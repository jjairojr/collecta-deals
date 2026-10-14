"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import Field from "@/components/checkout/Field";
import { PasswordField } from "@/components/account/LoginForm";
import { useAccount } from "@/lib/account";
import { trackSignUp } from "@/lib/analytics";
import { validEmail } from "@/lib/checkout";
import { pixelText } from "@/lib/format";

type Check = { state: "idle" } | { state: "checking" } | { state: "ok"; owner: string } | { state: "bad"; reason: string };

type Errors = Partial<Record<"email" | "password" | "displayName" | "referralCode", string>>;

const CODE_RE = /^[ABCDEFGHJKLMNPQRSTUVWXYZ23456789]{6}$/;

export default function RegisterForm({ next, initialCode }: { next: string; initialCode: string }) {
  const router = useRouter();
  const { refresh } = useAccount();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [code, setCode] = useState(initialCode);
  const [check, setCheck] = useState<Check>({ state: "idle" });
  const [errors, setErrors] = useState<Errors>({});
  const [notice, setNotice] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    const value = code.trim().toUpperCase();
    if (!value) {
      setCheck({ state: "idle" });
      return;
    }
    if (!CODE_RE.test(value)) {
      setCheck({ state: "bad", reason: "Código inválido" });
      return;
    }
    let cancelled = false;
    setCheck({ state: "checking" });
    const t = setTimeout(() => {
      fetch(`/api/conta/indicacao?code=${value}`, { cache: "no-store" })
        .then((res) => res.json())
        .then((data: { ok: boolean; ownerUsername?: string; reason?: string }) => {
          if (cancelled) {
            return;
          }
          setCheck(data.ok ? { state: "ok", owner: data.ownerUsername ?? "" } : { state: "bad", reason: data.reason ?? "Código inválido" });
        })
        .catch(() => {
          if (!cancelled) {
            setCheck({ state: "idle" });
          }
        });
    }, 350);
    return () => {
      cancelled = true;
      clearTimeout(t);
    };
  }, [code]);

  const submit = async () => {
    const e: Errors = {};
    if (!validEmail(email)) {
      e.email = "E-mail inválido";
    }
    if (password.length < 8) {
      e.password = "Mínimo 8 caracteres";
    }
    if (check.state === "bad") {
      e.referralCode = check.reason;
    }
    setErrors(e);
    if (Object.keys(e).length > 0) {
      return;
    }
    setSubmitting(true);
    setNotice(null);
    try {
      const res = await fetch("/api/conta/criar", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password, displayName, referralCode: code.trim().toUpperCase() }),
      });
      const data = (await res.json()) as { error?: string };
      if (!res.ok) {
        setNotice(data.error ?? "Não foi possível criar a conta.");
        return;
      }
      trackSignUp(check.state === "ok");
      await refresh();
      router.push(next);
    } catch {
      setNotice("Sem conexão. Tente de novo em instantes.");
    } finally {
      setSubmitting(false);
    }
  };

  const loginHref = next === "/conta" ? "/conta/entrar" : `/conta/entrar?next=${encodeURIComponent(next)}`;

  return (
    <form
      className="grid gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void submit();
      }}
    >
      <Field
        label="Nome"
        name="displayName"
        value={displayName}
        onChange={setDisplayName}
        autoComplete="name"
        maxLength={40}
      />
      <Field
        label="E-mail"
        name="email"
        value={email}
        onChange={(v) => {
          setEmail(v);
          setErrors((x) => ({ ...x, email: undefined }));
        }}
        error={errors.email}
        inputMode="email"
        autoComplete="email"
      />
      <PasswordField
        label="Senha"
        value={password}
        onChange={(v) => {
          setPassword(v);
          setErrors((x) => ({ ...x, password: undefined }));
        }}
        error={errors.password}
        autoComplete="new-password"
      />
      <div>
        <Field
          label="Código de indicação (opcional)"
          name="referralCode"
          value={code}
          onChange={(v) => {
            setCode(v.toUpperCase());
            setErrors((x) => ({ ...x, referralCode: undefined }));
          }}
          error={errors.referralCode}
          maxLength={6}
          placeholder="ABC234"
        />
        {check.state === "checking" && (
          <p className="mt-1 font-pixel text-[7px] text-brand-soft animate-blink">CONFERINDO...</p>
        )}
        {check.state === "ok" && (
          <p className="mt-1 font-pixel text-[7px] text-brand-soft">{pixelText(`INDICADO POR @${check.owner}`)}</p>
        )}
        {check.state === "bad" && !errors.referralCode && (
          <p className="mt-1 font-pixel text-[7px] text-brand">{pixelText(check.reason)}</p>
        )}
      </div>
      {notice && (
        <p role="alert" className="sticker rounded-[10px] bg-brand-soft px-3 py-2 text-xs font-semibold text-outline [--sh:3px]">
          {notice}
        </p>
      )}
      <button
        type="submit"
        disabled={submitting || check.state === "checking"}
        className="arcade-press sticker mt-1 w-full rounded-[10px] bg-brand px-5 py-3.5 font-pixel text-[10px] text-white [--sh:5px] disabled:cursor-wait disabled:opacity-60"
      >
        {submitting ? "CRIANDO..." : "CRIAR CONTA"}
      </button>
      <p className="text-center text-xs text-muted">
        Já tem conta no app?{" "}
        <Link href={loginHref} className="text-brand-soft underline-offset-4 hover:underline">
          Entrar
        </Link>
      </p>
    </form>
  );
}
