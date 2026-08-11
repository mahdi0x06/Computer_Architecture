module ALU(
    input wire clk,
    input wire rst,
    input wire [1:0] ALUOp,
    input wire alu_start_pulse,
    input wire [31:0] Reg1, Reg2,
    output reg [31:0] Result,
    output wire zero,
    output wire ready
);
    wire [31:0] mul_res, div_res;
    wire mul_ready, div_ready;

    wire start_mul = (ALUOp == 2'b10) && alu_start_pulse;
    wire start_div = (ALUOp == 2'b11) && alu_start_pulse;

    MUL mul(clk, start_mul, Reg1, Reg2, mul_res, mul_ready);
    DIV div(clk, start_div, Reg1, Reg2, div_res, div_ready);

    always @(*) begin
        case (ALUOp)
            2'b00: Result = Reg1 + Reg2;
            2'b01: Result = Reg1 - Reg2;
            2'b10: Result = mul_res;
            2'b11: Result = div_res;
            default: Result = 32'b0;
        endcase
    end
    
    assign zero = (Result == 32'b0) ? 1'b1 : 1'b0;
    assign ready = (ALUOp == 2'b10) ? mul_ready :
                   (ALUOp == 2'b11) ? div_ready : 1'b1;
endmodule

module MUL(
    input wire clk,
    input wire start,
    input wire [31:0] Reg1, Reg2,
    output reg [31:0] Result,
    output wire ready
);
    reg [5:0] count;
    reg [31:0] a, b;
    reg busy;
    
    initial busy = 1'b0;
    
    assign ready = ~busy & ~start;

    always @(posedge clk) begin
        if (start) begin
            a <= Reg1;
            b <= Reg2;
            Result <= 32'b0;
            count <= 6'b0;
            busy <= 1'b1;
        end else if (busy) begin
            if (a[0]) Result <= Result + b;
            a <= a >> 1;
            b <= b << 1;
            count <= count + 1;
            if (count == 31) busy <= 1'b0;
        end
    end
endmodule

module DIV(
    input wire clk,
    input wire start,
    input wire [31:0] Reg1, Reg2,
    output reg [31:0] Result,
    output wire ready
);
    reg [5:0] count;
    reg [31:0] q, r, b;
    reg busy;

    initial busy = 1'b0;

    assign ready = ~busy & ~start;

    always @(posedge clk) begin
        if (start) begin
            q <= Reg1;
            b <= Reg2;
            r <= 32'b0;
            Result <= 32'b0;
            count <= 32;
            busy <= 1'b1;
        end else if (busy) begin
            if ({r[30:0], q[31]} >= b) begin
                r <= {r[30:0], q[31]} - b;
                Result[count - 1] <= 1'b1;
            end else begin
                r <= {r[30:0], q[31]};
                Result[count - 1] <= 1'b0;
            end
            q <= q << 1;
            count <= count - 1;
            if (count == 1) busy <= 1'b0;
        end
    end
endmodule

module CU(
    input wire [5:0] opcode, funct,
    output reg [1:0] RegDst, MemtoReg, ALUOp,
    output reg RegWrite, ALUSrc, MemRead, MemWrite, Branch, Jump, Jal, Jr
);
    always @(*) begin
        RegDst = 2'b00;
        MemtoReg = 2'b00;
        ALUOp = 2'b00;
        RegWrite = 1'b0;
        ALUSrc = 1'b0;
        MemRead = 1'b0;
        MemWrite = 1'b0;
        Branch = 1'b0;
        Jump = 1'b0;
        Jal = 1'b0;
        Jr = 1'b0;

        case(opcode) 
            6'b000000:
                if(funct == 6'b001000) Jr = 1'b1;
                else begin
                    RegWrite = 1'b1;
                    case(funct)
                        6'b100000: ALUOp = 2'b00;
                        6'b100010: ALUOp = 2'b01;
                        6'b011000: ALUOp = 2'b10;
                        6'b011010: ALUOp = 2'b11;
                        default: ; 
                    endcase
                end
            
            6'b100011: begin ALUSrc = 1'b1; MemtoReg = 2'b01; MemRead = 1'b1; ALUOp = 2'b00; RegWrite = 1'b1; RegDst = 2'b01; end
            6'b101011: begin ALUSrc = 1'b1; MemWrite = 1'b1; ALUOp = 2'b00; RegDst = 2'b01; end
            6'b001000: begin ALUSrc = 1'b1; ALUOp = 2'b00; RegWrite = 1'b1; RegDst = 2'b01; end
            6'b001001: begin ALUSrc = 1'b1; ALUOp = 2'b01; RegWrite = 1'b1; RegDst = 2'b01; end
            6'b001111: begin ALUSrc = 1'b1; RegWrite = 1'b1; RegDst = 2'b01; MemtoReg = 2'b11; end
            6'b000010: begin Jump = 1'b1; end
            6'b000011: begin MemtoReg = 2'b10; Jump = 1'b1; Jal = 1'b1; RegWrite = 1'b1; RegDst = 2'b10; end
            6'b000100: begin Branch = 1'b1; ALUOp = 2'b01; end

            default: ; 
        endcase
    end
endmodule

module RF(
    input wire Clk, Rst, RegWrite,   
    input wire [4:0] ReadReg1, ReadReg2, WriteReg,
    input wire [31:0] Data,
    output wire [31:0] ReadData1, ReadData2,
    output wire [31:0] R0, R1, R2, R3, R4, R5, R6,
    R7, R8, R9, R10, R11, R12, R13, R14, R15,
    R16, R17, R18, R19, R20, R21, R22, R23, 
    R24, R25, R26, R27, R28, R29, R30, R31
);
    reg [31:0] Registers [0:31]; 
    integer i;

    initial begin
        for (i = 0; i < 32; i = i + 1) begin
            Registers[i] = 32'b0;
        end
    end

    assign ReadData1 = (ReadReg1 == 5'd0) ? 32'b0 : 
                       (RegWrite && WriteReg == ReadReg1) ? Data : Registers[ReadReg1];
    assign ReadData2 = (ReadReg2 == 5'd0) ? 32'b0 : 
                       (RegWrite && WriteReg == ReadReg2) ? Data : Registers[ReadReg2];
    
    assign R0  = 32'b0;
    assign R1  = (RegWrite && WriteReg == 5'd1)  ? Data : Registers[1];
    assign R2  = (RegWrite && WriteReg == 5'd2)  ? Data : Registers[2];
    assign R3  = (RegWrite && WriteReg == 5'd3)  ? Data : Registers[3];
    assign R4  = (RegWrite && WriteReg == 5'd4)  ? Data : Registers[4];
    assign R5  = (RegWrite && WriteReg == 5'd5)  ? Data : Registers[5];
    assign R6  = (RegWrite && WriteReg == 5'd6)  ? Data : Registers[6];
    assign R7  = (RegWrite && WriteReg == 5'd7)  ? Data : Registers[7];
    assign R8  = (RegWrite && WriteReg == 5'd8)  ? Data : Registers[8];
    assign R9  = (RegWrite && WriteReg == 5'd9)  ? Data : Registers[9];
    assign R10 = (RegWrite && WriteReg == 5'd10) ? Data : Registers[10];
    assign R11 = (RegWrite && WriteReg == 5'd11) ? Data : Registers[11];
    assign R12 = (RegWrite && WriteReg == 5'd12) ? Data : Registers[12];
    assign R13 = (RegWrite && WriteReg == 5'd13) ? Data : Registers[13];
    assign R14 = (RegWrite && WriteReg == 5'd14) ? Data : Registers[14];
    assign R15 = (RegWrite && WriteReg == 5'd15) ? Data : Registers[15];
    assign R16 = (RegWrite && WriteReg == 5'd16) ? Data : Registers[16];
    assign R17 = (RegWrite && WriteReg == 5'd17) ? Data : Registers[17];
    assign R18 = (RegWrite && WriteReg == 5'd18) ? Data : Registers[18];
    assign R19 = (RegWrite && WriteReg == 5'd19) ? Data : Registers[19];
    assign R20 = (RegWrite && WriteReg == 5'd20) ? Data : Registers[20];
    assign R21 = (RegWrite && WriteReg == 5'd21) ? Data : Registers[21];
    assign R22 = (RegWrite && WriteReg == 5'd22) ? Data : Registers[22];
    assign R23 = (RegWrite && WriteReg == 5'd23) ? Data : Registers[23];
    assign R24 = (RegWrite && WriteReg == 5'd24) ? Data : Registers[24];
    assign R25 = (RegWrite && WriteReg == 5'd25) ? Data : Registers[25];
    assign R26 = (RegWrite && WriteReg == 5'd26) ? Data : Registers[26];
    assign R27 = (RegWrite && WriteReg == 5'd27) ? Data : Registers[27];
    assign R28 = (RegWrite && WriteReg == 5'd28) ? Data : Registers[28];
    assign R29 = (RegWrite && WriteReg == 5'd29) ? Data : Registers[29];
    assign R30 = (RegWrite && WriteReg == 5'd30) ? Data : Registers[30];
    assign R31 = (RegWrite && WriteReg == 5'd31) ? Data : Registers[31];

    always @(posedge Clk or posedge Rst) begin
        if (Rst == 1'b1) begin
            for (i = 0; i < 32; i = i + 1) begin
                Registers[i] <= 32'b0;
            end
        end else if (RegWrite == 1'b1 && WriteReg != 5'b00000) begin 
            Registers[WriteReg] <= Data;
        end
    end
endmodule

module main (
    input wire Clk, Rst,
    
    input wire [31:0] InstrIn,
    output wire [31:0] InstrAddr,
    
    input wire [31:0] DataIn,
    output wire [31:0] DataOut,
    output wire DataWrite,
    output wire [31:0] DataAddr,

    output wire [31:0] D_PC, D_Instr,
    output wire D_Valid,
    output wire [31:0] D_Rs,
    output wire D_RsValid,
    output wire [31:0] D_Rt,
    output wire D_RtValid,
    output wire [15:0] D_Imm,
    output wire D_ImmValid,
    output wire [25:0] D_Address,
    output wire D_AddressValid,

    output wire [31:0] E_PC, E_Instr,
    output wire E_Valid,
    output wire [31:0] E_Res,
    output wire E_ResValid,

    output wire [31:0] W_PC,
    output wire W_Valid,
    
    output wire [31:0] R0, R1, R2, R3, R4, R5, R6, R7, R8, R9, R10, R11, R12, R13, R14, R15,
    output wire [31:0] R16, R17, R18, R19, R20, R21, R22, R23, R24, R25, R26, R27, R28, R29, R30, R31
);

    reg [31:0] PC_reg;
    wire [31:0] PC_plus_4 = PC_reg + 4;
    wire [31:0] Next_PC;

    assign InstrAddr = (PC_reg === 32'bx) ? 32'b0 : (PC_reg >> 2);

    reg [31:0] F_D_PC;
    reg [31:0] F_D_Instr;
    reg F_D_Valid;

    reg [31:0] D_E_PC, D_E_Instr, D_E_RsVal, D_E_RtVal, D_E_Imm;
    reg [4:0] D_E_Rs, D_E_Rt, D_E_Rd;
    reg D_E_Valid;
    reg [1:0] D_E_RegDst, D_E_MemtoReg, D_E_ALUOp;
    reg D_E_RegWrite, D_E_ALUSrc, D_E_MemRead, D_E_MemWrite, D_E_Jal;

    reg [31:0] E_M_PC, E_M_Instr, E_M_ALUOut, E_M_WriteData;
    reg [4:0] E_M_WriteReg;
    reg E_M_Valid;
    reg [1:0] E_M_MemtoReg;
    reg E_M_RegWrite, E_M_MemRead, E_M_MemWrite;

    reg [31:0] M_W_PC, M_W_Instr, M_W_Data_to_Reg;
    reg [4:0] M_W_WriteReg;
    reg M_W_Valid;
    reg M_W_RegWrite;

    wire [5:0] opcode = F_D_Instr[31:26];
    wire [4:0] rs = F_D_Instr[25:21];
    wire [4:0] rt = F_D_Instr[20:16];
    wire [4:0] rd = F_D_Instr[15:11];
    wire [5:0] funct = F_D_Instr[5:0];
    wire [15:0] imm = F_D_Instr[15:0];
    wire [25:0] addr = F_D_Instr[25:0];
    wire [31:0] sign_ext_imm = {{16{imm[15]}}, imm};

    wire [1:0] RegDst, MemtoReg, ALUOp;
    wire RegWrite, ALUSrc, MemRead, MemWrite, Branch, Jump, Jal, Jr;

    CU control_unit (
        .opcode(opcode),
        .funct(funct),
        .RegDst(RegDst),
        .MemtoReg(MemtoReg),
        .ALUOp(ALUOp),
        .RegWrite(RegWrite),
        .ALUSrc(ALUSrc),
        .MemRead(MemRead),
        .MemWrite(MemWrite),
        .Branch(Branch),
        .Jump(Jump),
        .Jal(Jal),
        .Jr(Jr)
    );

    wire [31:0] rf_read_data1, rf_read_data2;

    RF register_file (
        .Clk(Clk),
        .Rst(Rst),
        .RegWrite(M_W_RegWrite && M_W_Valid),
        .ReadReg1(rs),
        .ReadReg2(rt),
        .WriteReg(M_W_WriteReg),
        .Data(M_W_Data_to_Reg),
        .ReadData1(rf_read_data1),
        .ReadData2(rf_read_data2),
        .R0(R0), .R1(R1), .R2(R2), .R3(R3), .R4(R4), .R5(R5), .R6(R6), .R7(R7),
        .R8(R8), .R9(R9), .R10(R10), .R11(R11), .R12(R12), .R13(R13), .R14(R14), .R15(R15),
        .R16(R16), .R17(R17), .R18(R18), .R19(R19), .R20(R20), .R21(R21), .R22(R22), .R23(R23),
        .R24(R24), .R25(R25), .R26(R26), .R27(R27), .R28(R28), .R29(R29), .R30(R30), .R31(R31)
    );

    wire [31:0] M_Data_to_Reg = (E_M_MemtoReg == 2'b11) ? {E_M_Instr[15:0], 16'b0} :
                                (E_M_MemtoReg == 2'b10) ? ((E_M_PC << 2) + 4) :
                                (E_M_MemtoReg == 2'b01) ? DataIn :
                                E_M_ALUOut;

    wire [31:0] fw_rs_D = (E_M_Valid && E_M_RegWrite && E_M_WriteReg != 0 && E_M_WriteReg == rs) ? M_Data_to_Reg :
                          (M_W_Valid && M_W_RegWrite && M_W_WriteReg != 0 && M_W_WriteReg == rs) ? M_W_Data_to_Reg :
                          rf_read_data1;

    wire [31:0] fw_rt_D = (E_M_Valid && E_M_RegWrite && E_M_WriteReg != 0 && E_M_WriteReg == rt) ? M_Data_to_Reg :
                          (M_W_Valid && M_W_RegWrite && M_W_WriteReg != 0 && M_W_WriteReg == rt) ? M_W_Data_to_Reg :
                          rf_read_data2;

    wire D_Rs_used = (opcode == 6'b000000) || (opcode == 6'b001000) || (opcode == 6'b001001) || (opcode == 6'b100011) || (opcode == 6'b101011) || (opcode == 6'b000100);
    wire D_Rt_used = (opcode == 6'b000000) || (opcode == 6'b101011) || (opcode == 6'b000100);
    wire Branch_Rs_used = (opcode == 6'b000100) || (opcode == 6'b000000 && funct == 6'b001000);
    wire Branch_Rt_used = (opcode == 6'b000100);

    wire [4:0] D_E_WriteReg = (D_E_RegDst == 2'b10) ? 5'd31 : (D_E_RegDst == 2'b01) ? D_E_Rt : D_E_Rd;

    wire Stall_LoadUse = F_D_Valid && D_E_Valid && D_E_MemRead && D_E_WriteReg != 0 &&
                         ((D_Rs_used && D_E_WriteReg == rs) || (D_Rt_used && D_E_WriteReg == rt));

    wire Stall_Branch = F_D_Valid && D_E_Valid && D_E_RegWrite && D_E_WriteReg != 0 &&
                        ((Branch_Rs_used && D_E_WriteReg == rs) || (Branch_Rt_used && D_E_WriteReg == rt));

    wire alu_ready;
    wire Stall_ALU = D_E_Valid && !alu_ready;

    wire Stall_F_D = Stall_ALU || Stall_LoadUse || Stall_Branch;

    wire BranchTaken = F_D_Valid && Branch && (fw_rs_D == fw_rt_D);
    wire JumpTaken = F_D_Valid && Jump;
    wire JrTaken = F_D_Valid && Jr;

    wire Flush_F = !Stall_F_D && (BranchTaken || JumpTaken || JrTaken);

    wire [31:0] f_d_pc_plus_4 = (F_D_PC << 2) + 4;
    wire [31:0] branch_target = f_d_pc_plus_4 + (sign_ext_imm << 2);
    wire [31:0] jump_target = {f_d_pc_plus_4[31:28], addr, 2'b00};

    assign Next_PC = JrTaken ? fw_rs_D :
                     BranchTaken ? branch_target :
                     JumpTaken ? jump_target :
                     PC_plus_4;

    always @(posedge Clk or posedge Rst) begin
        if (Rst) PC_reg <= 32'd0;
        else if (!Stall_F_D) PC_reg <= Next_PC;
    end

    always @(posedge Clk or posedge Rst) begin
        if (Rst) begin
            F_D_PC <= 0;
            F_D_Instr <= 0;
            F_D_Valid <= 0;
        end else if (!Stall_F_D) begin
            if (Flush_F) begin
                F_D_Valid <= 0;
                F_D_Instr <= 0;
            end else begin
                F_D_PC <= PC_reg >> 2;
                F_D_Instr <= InstrIn;
                F_D_Valid <= 1'b1;
            end
        end
    end

    always @(posedge Clk or posedge Rst) begin
        if (Rst) begin
            D_E_Valid <= 0;
            D_E_RegWrite <= 0;
            D_E_MemWrite <= 0;
            D_E_MemRead <= 0;
        end else if (Stall_ALU) begin
            
        end else if (Stall_LoadUse || Stall_Branch) begin
            D_E_Valid <= 0;
            D_E_RegWrite <= 0;
            D_E_MemWrite <= 0;
            D_E_MemRead <= 0;
        end else begin
            D_E_PC <= F_D_PC;
            D_E_Instr <= F_D_Instr;
            D_E_Valid <= F_D_Valid;
            D_E_RsVal <= fw_rs_D;
            D_E_RtVal <= fw_rt_D;
            D_E_Imm <= sign_ext_imm;
            D_E_Rs <= rs;
            D_E_Rt <= rt;
            D_E_Rd <= rd;
            D_E_RegDst <= RegDst;
            D_E_MemtoReg <= MemtoReg;
            D_E_ALUOp <= ALUOp;
            D_E_RegWrite <= RegWrite;
            D_E_ALUSrc <= ALUSrc;
            D_E_MemRead <= MemRead;
            D_E_MemWrite <= MemWrite;
            D_E_Jal <= Jal;
        end
    end

    wire [1:0] fw_rs_E = (E_M_Valid && E_M_RegWrite && E_M_WriteReg != 0 && E_M_WriteReg == D_E_Rs) ? 2'b10 :
                         (M_W_Valid && M_W_RegWrite && M_W_WriteReg != 0 && M_W_WriteReg == D_E_Rs) ? 2'b01 : 2'b00;

    wire [1:0] fw_rt_E = (E_M_Valid && E_M_RegWrite && E_M_WriteReg != 0 && E_M_WriteReg == D_E_Rt) ? 2'b10 :
                         (M_W_Valid && M_W_RegWrite && M_W_WriteReg != 0 && M_W_WriteReg == D_E_Rt) ? 2'b01 : 2'b00;

    wire [31:0] ALU_in1 = (fw_rs_E == 2'b10) ? M_Data_to_Reg :
                          (fw_rs_E == 2'b01) ? M_W_Data_to_Reg : D_E_RsVal;

    wire [31:0] ALU_in2_temp = (fw_rt_E == 2'b10) ? M_Data_to_Reg :
                               (fw_rt_E == 2'b01) ? M_W_Data_to_Reg : D_E_RtVal;

    wire [31:0] ALU_in2 = D_E_ALUSrc ? D_E_Imm : ALU_in2_temp;

    wire [31:0] alu_result;
    wire alu_zero;

    reg alu_active;
    always @(posedge Clk or posedge Rst) begin  
        if (Rst) alu_active <= 0;
        else if (Stall_ALU) alu_active <= 1;
        else alu_active <= 0;
    end
    wire alu_start_pulse = D_E_Valid && !alu_active && (D_E_ALUOp == 2'b10 || D_E_ALUOp == 2'b11);

    ALU alu_inst (
        .clk(Clk),
        .rst(Rst),
        .ALUOp(D_E_ALUOp),
        .alu_start_pulse(alu_start_pulse),
        .Reg1(ALU_in1),
        .Reg2(ALU_in2),
        .Result(alu_result),
        .zero(alu_zero),
        .ready(alu_ready)
    );

    always @(posedge Clk or posedge Rst) begin
        if (Rst) begin
            E_M_Valid <= 0;
            E_M_RegWrite <= 0;
            E_M_MemWrite <= 0;
            E_M_MemRead <= 0;
        end else if (Stall_ALU) begin
            E_M_Valid <= 0;
            E_M_RegWrite <= 0;
            E_M_MemWrite <= 0;
            E_M_MemRead <= 0;
        end else begin
            E_M_PC <= D_E_PC;
            E_M_Instr <= D_E_Instr;
            E_M_Valid <= D_E_Valid;
            E_M_ALUOut <= alu_result;
            E_M_WriteData <= ALU_in2_temp; 
            E_M_WriteReg <= D_E_WriteReg;
            E_M_MemtoReg <= D_E_MemtoReg;
            E_M_RegWrite <= D_E_RegWrite;
            E_M_MemRead <= D_E_MemRead;
            E_M_MemWrite <= D_E_MemWrite;
        end
    end

    assign DataAddr = (E_M_Valid && (E_M_MemRead || E_M_MemWrite)) ? (E_M_ALUOut >> 2) : 32'h00004000;
    assign DataWrite = E_M_MemWrite && E_M_Valid;
    assign DataOut = E_M_WriteData;

    always @(posedge Clk or posedge Rst) begin
        if (Rst) begin
            M_W_Valid <= 0;
            M_W_RegWrite <= 0;
        end else begin
            M_W_PC <= E_M_PC;
            M_W_Instr <= E_M_Instr;
            M_W_Valid <= E_M_Valid;
            M_W_Data_to_Reg <= M_Data_to_Reg;
            M_W_WriteReg <= E_M_WriteReg;
            M_W_RegWrite <= E_M_RegWrite;
        end
    end

    assign D_PC = F_D_PC;
    assign D_Instr = F_D_Instr;
    assign D_Valid = F_D_Valid;
    assign D_Rs = fw_rs_D;
    assign D_RsValid = F_D_Valid;
    assign D_Rt = fw_rt_D;
    assign D_RtValid = F_D_Valid;
    assign D_Imm = imm;
    assign D_ImmValid = F_D_Valid;
    assign D_Address = addr;
    assign D_AddressValid = F_D_Valid;

    assign E_PC = D_E_PC;
    assign E_Instr = D_E_Instr;
    assign E_Valid = D_E_Valid;
    assign E_Res = alu_result;
    assign E_ResValid = D_E_Valid;

    assign W_PC = M_W_PC;
    assign W_Valid = M_W_Valid;

endmodule