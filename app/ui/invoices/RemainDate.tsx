"use client"

import { useEffect, useState } from "react";
import { formatRemainDate, formatDateToLocal } from "@/app/lib/utils";

export default function RemainDate({ timestamp, locale }: { timestamp: number, locale?: string }) {

    const [remain, setRemain] = useState(formatRemainDate(timestamp, locale));
    useEffect(() => {
        const timer = setInterval(() => {
            setRemain(formatRemainDate(timestamp, locale));
        }, 1000);

        return () => { clearInterval(timer) }
    })

    const colorClass = remain.hour > 0 ? remain.hour > 2 ? 'bg-gray-100 text-gray-500' : 'bg-orange-100 text-orange-500' : 'bg-rose-100 text-rose-500'
    return (<>
        <span className={`${colorClass} font-medium p-1 m-1 rounded`}>{remain.hour.toString().padStart(2, '0')}</span>:
        <span className={`${colorClass} font-medium p-1 m-1 rounded`}>{remain.minute.toString().padStart(2, '0')}</span>:
        <span className={`${colorClass} font-medium p-1 m-1 rounded`}>{remain.second.toString().padStart(2, '0')}</span>
    </>)
}
