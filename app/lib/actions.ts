"use server";

import { z } from "zod";
import { client } from "@/app/lib/db/postgres";
import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";

import { signIn } from "@/auth";
import { AuthError } from "next-auth";

const AuctionFormSchema = z.object({
  id: z.coerce.number(),
  price: z.coerce.number(),
  address: z.string(),
  status: z.enum(["listened", "started", "paused", "stoped"]),
});

const CreateAuction = AuctionFormSchema.omit({ price: true, address: true });
export async function createAuction(formData: FormData): Promise<void> {
  const rawFormData = Object.fromEntries(formData.entries());

  const body = CreateAuction.parse(rawFormData);

  const auctions = await fetch("http://localhost:8080/auction", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
    redirect: "follow",
  });
  if (auctions.status == 200) {
    revalidatePath("/dashboard/invoices");
    redirect("/dashboard/invoices");
  }

  throw new Error("创建拍卖任务失败");
}

const FormSchema = z.object({
  id: z.string(),
  customerId: z.string(),
  amount: z.coerce.number(),
  status: z.enum(["pending", "paid"]),
  date: z.string(),
});

const CreateInvoice = FormSchema.omit({ id: true, date: true });

export async function createInvoice(formData: FormData): Promise<void> {
  //   const rawFormData = {
  //     customerId: formData.get("customerId"),
  //     amount: formData.get("amount"),
  //     status: formData.get("status"),
  //   };

  const rawFormData = Object.fromEntries(formData.entries());

  const { customerId, amount, status } = CreateInvoice.parse(rawFormData);
  const amountInCents = amount * 100;
  const date = new Date().toISOString().split("T")[0];

  try {
    await client.sql`
    INSERT INTO invoices (customer_id, amount, status, date)
    VALUES (${customerId}, ${amountInCents}, ${status}, ${date})
  `;
    console.log(rawFormData);
  } catch (error) {
    return {
      message: "Database Error: Failed to Create Invoice.",
    };
  }

  revalidatePath("/dashboard/invoices");
  redirect("/dashboard/invoices");
}

const UpdateInvoice = FormSchema.omit({ id: true, date: true });

export async function updateInvoice(id: string, formData: FormData) {
  const rawFormData = {
    customerId: formData.get("customerId"),
    amount: formData.get("amount"),
    status: formData.get("status"),
  };

  const { customerId, amount, status } = CreateInvoice.parse(rawFormData);
  const amountInCents = amount * 100;

  try {
    await client.sql`
    UPDATE invoices
    SET customer_id = ${customerId}, amount = ${amountInCents}, status = ${status}
    WHERE id = ${id}
  `;
  } catch (error) {
    return {
      message: "Database Error: Failed to Update Invoice.",
    };
  }

  revalidatePath("/dashboard/invoices");
  redirect("/dashboard/invoices");
}

export async function deleteInvoice(id: string) {
  throw new Error("Failed to Delete Invoice");
  try {
    await client.sql`DELETE FROM invoices WHERE id = ${id}`;
  } catch (error) {
    return {
      message: "Database Error: Failed to Delete Invoice.",
    };
  }

  revalidatePath("/dashboard/invoices");
  return { message: "Deleted Invoice." };
}

export async function authenticate(
  prevState: string | undefined,
  formData: FormData
) {
  try {
    await signIn("credentials", formData);
  } catch (error) {
    if (error instanceof AuthError) {
      switch (error.type) {
        case "CredentialsSignin":
          return "Invalid credentials.";
        default:
          return "Something went wrong.";
      }
    }
    throw error;
  }
}
