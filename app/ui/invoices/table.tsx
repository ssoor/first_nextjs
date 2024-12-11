import Image from "next/image";
import {
  UpdateInvoice,
  DeleteInvoice,
  CreateInvoice,
} from "@/app/ui/invoices/buttons";
import InvoiceStatus from "@/app/ui/invoices/status";
import { formatDateToLocal, formatCurrency } from "@/app/lib/utils";
import { fetchFilteredInvoices, fetchAuctions } from "@/app/lib/data";
import RemainDate from "./RemainDate";

export default async function InvoicesTable({
  query,
  status,
  currentPage,
}: {
  query: string;
  status: string;
  currentPage: number;
}) {
  const auctions = await fetchAuctions(query, status, currentPage);

  return (
    <div className="mt-6 flow-root">
      <div className="inline-block min-w-full align-middle">
        <div className="rounded-lg bg-gray-50 p-2 md:pt-0">
          <div className="md:hidden">
            {auctions?.map((auction) => (
              <div
                key={auction.id}
                className="mb-2 w-full rounded-md bg-white p-4"
              >
                <div className="flex items-center justify-between border-b pb-4">
                  <div>
                    <div className="mb-2 flex items-center">
                      <Image
                        src={auction.primary_pic}
                        className="mr-2 rounded-full"
                        width={28}
                        height={28}
                        alt={`${auction.name}'s profile picture`}
                      />
                      <p>{auction.name}</p>
                    </div>
                    <p className="text-sm text-gray-500">{`${auction.quality}`}</p>
                  </div>
                  <InvoiceStatus status={auction.status} />
                </div>
                <div className="flex w-full items-center justify-between pt-4">
                  <div>
                    <p className="text-xl font-medium">
                      {formatCurrency(auction.current_price)}
                    </p>
                    <p>{formatDateToLocal(auction.end_timestamp)}</p>
                  </div>
                  <div className="flex justify-end gap-2">
                    <CreateInvoice id={"1"} />
                    <UpdateInvoice id={"1"} />
                    <DeleteInvoice id={"1"} />
                  </div>
                </div>
              </div>
            ))}
          </div>
          <table className="hidden min-w-full text-gray-900 md:table">
            <thead className="rounded-lg text-left text-sm font-normal">
              <tr>
                <th scope="col" className="px-4 py-5 font-medium sm:pl-6">
                  商品
                </th>
                <th scope="col" className="px-3 py-5 font-medium">
                  质量
                </th>
                <th scope="col" className="px-3 py-5 font-medium">
                  当前价
                </th>
                <th scope="col" className="px-3 py-5 font-medium">
                  距离结束
                </th>
                <th scope="col" className="px-3 py-5 font-medium">
                  状态
                </th>
                <th scope="col" className="relative py-3 pl-6 pr-3">
                  <span className="sr-only">操作</span>
                </th>
              </tr>
            </thead>
            <tbody className="bg-white">
              {auctions?.map((auction) => (
                <tr
                  key={auction.id}
                  className="w-full border-b py-3 text-sm last-of-type:border-none [&:first-child>td:first-child]:rounded-tl-lg [&:first-child>td:last-child]:rounded-tr-lg [&:last-child>td:first-child]:rounded-bl-lg [&:last-child>td:last-child]:rounded-br-lg"
                >
                  <td className="whitespace-nowrap py-3 pl-6 pr-3">
                    <div className="flex items-center gap-3">
                      <Image
                        src={auction.primary_pic}
                        // className="rounded-full"
                        width={100}
                        height={100}
                        alt={`${auction.name}'s profile picture`}
                      />
                      <p>{auction.name.substring(0, 32)}</p>
                    </div>
                  </td>
                  <td className="whitespace-nowrap px-3 py-3">
                    {`${auction.quality}`}
                  </td>
                  <td className="whitespace-nowrap px-3 py-3">
                    {formatCurrency(auction.current_price)}
                  </td>
                  <td className="whitespace-nowrap px-3 py-3">
                    <RemainDate timestamp={auction.end_timestamp} />
                  </td>
                  <td className="whitespace-nowrap px-3 py-3">
                    <InvoiceStatus status={auction.status} />
                  </td>
                  <td className="whitespace-nowrap py-3 pl-6 pr-3">
                    <div className="flex justify-end gap-3">
                      {auction.listened ? (
                        <>
                          <UpdateInvoice id={auction.id} />
                          <DeleteInvoice id={auction.id} />
                        </>
                      ) : (
                        <CreateInvoice id={auction.id} />
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
